//! near-stocks core settlement contract (Development.md §5, behavior-spec).
//! Phase 3: endpoint, spot collateral, perps, exchange, clearinghouse, PERPTICK with price guards,
//! ft_on_transfer deposits, WITHDRAW_COLLATERAL with re-credit, NEP-297 events, admin/pause/upgrade,
//! and the migrate_state latch. Options, pre-market and synthetic spot are Phase 4. This deployment
//! carries no LogX token, staking or rewards pool.
pub mod eip712;
pub mod events;
pub mod fixed;
pub mod identity;
pub mod perp;
pub mod products;
pub mod risk;
pub mod state;
pub mod tx;

use ethnum::I256;
use events::{emit, hex, unhex20, unhex32};
use fixed::*;
use near_sdk::borsh::{self, BorshDeserialize, BorshSerialize};
use near_sdk::json_types::{I128, U128};
use near_sdk::serde_json::{json, Value};
use near_sdk::store::{IterableMap, LookupMap};
use near_sdk::{env, near, AccountId, BorshStorageKey, Gas, NearToken, PanicOnDefault, Promise, PromiseOrValue};
use perp::{FeeTable, PerpBalance};
use risk::Market;
use state::*;

const GAS_FT_TRANSFER: Gas = Gas::from_tgas(15);
const GAS_WITHDRAW_CALLBACK: Gas = Gas::from_tgas(10);
const ONE_YOCTO: NearToken = NearToken::from_yoctonear(1);
const DEFAULT_SESSION_TTL_MS: u64 = 7 * 24 * 3600 * 1000;
const DEFAULT_PRICE_MAX_AGE_SEC: u64 = 300;
/// db.ORDER_EXPIRY_DURATION is 30 days; one extra day covers clock skew. Bounds how long a
/// replay-protection entry must be kept before prune_filled may remove it.
const DEFAULT_MAX_ORDER_TTL_MS: u64 = 31 * 24 * 3600 * 1000;

#[derive(BorshStorageKey, BorshSerialize)]
#[borsh(crate = "near_sdk::borsh")]
enum Key {
    Spots,
    Perps,
    TokenToSpot,
    Subaccounts,
    Owners,
    SessionKeys,
    Nonces,
    Filled,
    SideProducts,
    Options,
    PoolBalances,
    HeldDeposits,
    Unclaimed,
}

/// Replay protection for signed orders (§5.6 step 2). `expiration_ms` lets the sequencer prune
/// entries once the order can no longer be replayed (storage, §5.11).
#[near(serializers = [borsh])]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Fill {
    pub filled: i128,
    pub expiration_ms: u64,
}

#[near(serializers = [json])]
#[derive(Clone)]
pub struct SpotConfig {
    pub token: Option<AccountId>,
    pub decimals: u8,
    pub weighted: bool,
    pub withdraw_fee_x18: I128,
    pub price_x18: I128,
    pub max_deviation_bps: u32,
}

#[near(serializers = [json])]
#[derive(Clone)]
pub struct PerpConfig {
    pub imf_x18: I128,
    pub mmf_x18: I128,
    pub liq_frac_x18: I128,
    pub price_x18: I128,
    pub max_deviation_bps: u32,
    /// largest absolute AMM position (base, x18); 0 = no cap. Must equal the backend's markets row.
    pub amm_max_position_x18: I128,
}

#[near(serializers = [json])]
#[derive(Clone)]
pub struct Limits {
    pub price_max_age_sec: u64,
    pub max_session_ttl_ms: u64,
    /// smallest deposit (x18) that may create a new subaccount (storage-spam guard)
    pub min_new_deposit_x18: I128,
    pub max_order_ttl_ms: u64,
}

/// One subaccount in a migrate_state chunk (§11). Values are x18 strings.
#[near(serializers = [json])]
pub struct MigrationEntry {
    pub subaccount: String,
    pub owner: Option<AccountId>,
    pub nonce: u64,
    pub spots: Vec<(u32, I128)>,
    /// (product id, amount, v_quote, last_cum_funding)
    pub perps: Vec<(u32, I128, I128, I128)>,
}

#[near(serializers = [json])]
pub struct PerpState {
    pub product_id: u32,
    pub cum_funding_x18: I128,
    pub last_funding_time: u64,
    pub open_interest_long: I128,
    pub open_interest_short: I128,
}

#[near(contract_state)]
#[derive(PanicOnDefault)]
pub struct NearStocks {
    owner: AccountId,
    guardian: AccountId,
    sequencer: AccountId,
    chain_id: u64,
    broker_id: u64,
    n_submissions: u64,
    paused: u8,
    migrated: bool,
    fee_table: FeeTable,
    amm_subaccount: [u8; 32],
    insurance_subaccount: [u8; 32],
    /// D-7: trading, liquidation-trade and withdrawal fees are credited here
    fee_subaccount: [u8; 32],
    price_max_age_sec: u64,
    max_session_ttl_ms: u64,
    min_new_deposit_x18: i128,
    max_order_ttl_ms: u64,
    /// ALL_SPOTS_ON_CONTRACT: settlement order for negative pnl
    spot_order: Vec<u32>,
    /// ALL_COLLATERAL_SPOTS: withdrawable walk (in reverse)
    collateral_order: Vec<u32>,
    spots: IterableMap<u32, SpotProduct>,
    perps: IterableMap<u32, PerpProduct>,
    token_to_spot: LookupMap<AccountId, u32>,
    subaccounts: LookupMap<[u8; 32], Subaccount>,
    owners: LookupMap<[u8; 32], AccountId>,
    session_keys: LookupMap<([u8; 32], [u8; 20]), u64>,
    nonces: LookupMap<[u8; 32], u64>,
    filled: LookupMap<[u8; 32], Fill>,
    product_accounts: ProductAccounts,
    side_products: IterableMap<(u8, u32), SideProduct>,
    options: LookupMap<u64, StoredBet>,
    /// pre-market (kind 1) and synthetic-spot (kind 2) token ledgers, keyed (kind, subaccount, pid)
    pool_balances: LookupMap<(u8, [u8; 32], u32), i128>,
    trusted_depositors: Vec<AccountId>,
    /// trusted deposits that arrived while deposits were closed, x18, keyed (account, subaccount number, product)
    held: LookupMap<(AccountId, u64, u32), i128>,
    /// trusted deposits whose msg could not be read, token units per product
    unclaimed: LookupMap<u32, u128>,
}

/// Prices and product config as seen by the risk functions. `overrides` carries the prices a
/// sequencer transaction supplied (liquidation, settlement); everything else is stored state.
struct ChainMarket<'a> {
    c: &'a NearStocks,
    now_sec: u64,
    overrides: &'a [(u32, i128)],
}

impl ChainMarket<'_> {
    fn override_price(&self, pid: u32) -> Option<i128> {
        self.overrides.iter().find(|(p, _)| *p == pid).map(|(_, v)| *v)
    }

    fn fresh(&self, p: &Price) -> bool {
        self.c.price_max_age_sec == 0 || self.now_sec.saturating_sub(p.updated_at) <= self.c.price_max_age_sec
    }
}

impl Market for ChainMarket<'_> {
    fn spot_price(&self, pid: u32) -> Option<i128> {
        if let Some(p) = self.override_price(pid) {
            return Some(p);
        }
        let sp = self.c.spots.get(&pid)?;
        if sp.price.updated_at == 0 || sp.price.price_x18 == 0 {
            return None;
        }
        if !self.fresh(&sp.price) {
            env::panic_str(&format!("stale price for product {pid}"));
        }
        Some(sp.price.price_x18)
    }

    fn spot_weighted(&self, pid: u32) -> bool {
        self.c.spots.get(&pid).map(|s| s.weighted).unwrap_or(false)
    }

    fn perp_price(&self, pid: u32) -> i128 {
        if let Some(p) = self.override_price(pid) {
            return p;
        }
        let pp = self.c.perps.get(&pid).unwrap_or_else(|| env::panic_str("unknown perp"));
        if pp.price.updated_at == 0 || pp.price.price_x18 <= 0 {
            env::panic_str(&format!("no price for product {pid}"));
        }
        if !self.fresh(&pp.price) {
            env::panic_str(&format!("stale price for product {pid}"));
        }
        pp.price.price_x18
    }

    fn perp_margins(&self, pid: u32) -> (i128, i128) {
        let pp = self.c.perps.get(&pid).unwrap_or_else(|| env::panic_str("unknown perp"));
        (pp.imf_x18, pp.mmf_x18)
    }

    fn cum_funding(&self, pid: u32) -> i128 {
        self.c.perps.get(&pid).map(|p| p.cum_funding_x18).unwrap_or(0)
    }
}

fn broker_of(sub: &[u8; 32]) -> u64 {
    let mut b = [0u8; 8];
    b[2..].copy_from_slice(&sub[0..6]);
    u64::from_be_bytes(b)
}

fn now_ms() -> u64 {
    env::block_timestamp_ms()
}

fn now_sec() -> u64 {
    env::block_timestamp() / 1_000_000_000
}

fn s(v: i128) -> String {
    v.to_string()
}

fn decode<T: BorshDeserialize>(payload: &[u8]) -> T {
    borsh::from_slice(payload).unwrap_or_else(|_| env::panic_str("invalid transaction payload"))
}

fn pow10(n: u8) -> i128 {
    10i128.checked_pow(n as u32).unwrap_or_else(|| env::panic_str("decimals out of range"))
}

/// The contract's on-chain field layout before the 2026-09-27 LogX/staking/rewards removal
/// (Development.md §16.11) — exists only so `migrate()` can decode already-deployed state. `pub`
/// so `core/tests` can exercise the migration directly with realistic data; delete both this and
/// `PreLogxRemovalClaimLimits` once `migrate()` has actually run against the deployed contract.
#[near(serializers = [borsh])]
pub struct PreLogxRemovalClaimLimits {
    pub max_logx_claim_x18: I128,
    pub max_reward_claim_x18: I128,
}

#[near(serializers = [borsh])]
pub struct PreLogxRemoval {
    pub owner: AccountId,
    pub guardian: AccountId,
    pub sequencer: AccountId,
    pub chain_id: u64,
    pub broker_id: u64,
    pub n_submissions: u64,
    pub paused: u8,
    pub migrated: bool,
    pub fee_table: FeeTable,
    pub amm_subaccount: [u8; 32],
    pub insurance_subaccount: [u8; 32],
    pub fee_subaccount: [u8; 32],
    pub price_max_age_sec: u64,
    pub max_session_ttl_ms: u64,
    pub min_new_deposit_x18: i128,
    pub max_order_ttl_ms: u64,
    pub spot_order: Vec<u32>,
    pub collateral_order: Vec<u32>,
    pub spots: IterableMap<u32, SpotProduct>,
    pub perps: IterableMap<u32, PerpProduct>,
    pub token_to_spot: LookupMap<AccountId, u32>,
    pub subaccounts: LookupMap<[u8; 32], Subaccount>,
    pub owners: LookupMap<[u8; 32], AccountId>,
    pub session_keys: LookupMap<([u8; 32], [u8; 20]), u64>,
    pub nonces: LookupMap<[u8; 32], u64>,
    pub filled: LookupMap<[u8; 32], Fill>,
    pub product_accounts: ProductAccounts,
    pub side_products: IterableMap<(u8, u32), SideProduct>,
    pub options: LookupMap<u64, StoredBet>,
    pub pool_balances: LookupMap<(u8, [u8; 32], u32), i128>,
    pub claim_limits: PreLogxRemovalClaimLimits,
    pub reward_rate_x18: i128,
    pub trusted_depositors: Vec<AccountId>,
    pub held: LookupMap<(AccountId, u64, u32), i128>,
    pub unclaimed: LookupMap<u32, u128>,
}

#[near]
impl NearStocks {
    #[init]
    pub fn new(
        owner: AccountId,
        guardian: AccountId,
        sequencer: AccountId,
        chain_id: u64,
        broker_id: u64,
        amm_subaccount: String,
        insurance_subaccount: String,
        fee_subaccount: String,
    ) -> Self {
        Self {
            owner,
            guardian,
            sequencer,
            chain_id,
            broker_id,
            n_submissions: 0,
            paused: 0,
            migrated: false,
            fee_table: FeeTable::legacy(),
            amm_subaccount: unhex32(&amm_subaccount),
            insurance_subaccount: unhex32(&insurance_subaccount),
            fee_subaccount: unhex32(&fee_subaccount),
            price_max_age_sec: DEFAULT_PRICE_MAX_AGE_SEC,
            max_session_ttl_ms: DEFAULT_SESSION_TTL_MS,
            min_new_deposit_x18: E18,
            max_order_ttl_ms: DEFAULT_MAX_ORDER_TTL_MS,
            spot_order: vec![],
            collateral_order: vec![],
            spots: IterableMap::new(Key::Spots),
            perps: IterableMap::new(Key::Perps),
            token_to_spot: LookupMap::new(Key::TokenToSpot),
            subaccounts: LookupMap::new(Key::Subaccounts),
            owners: LookupMap::new(Key::Owners),
            session_keys: LookupMap::new(Key::SessionKeys),
            nonces: LookupMap::new(Key::Nonces),
            filled: LookupMap::new(Key::Filled),
            product_accounts: ProductAccounts::default(),
            side_products: IterableMap::new(Key::SideProducts),
            options: LookupMap::new(Key::Options),
            pool_balances: LookupMap::new(Key::PoolBalances),
            trusted_depositors: vec![],
            held: LookupMap::new(Key::HeldDeposits),
            unclaimed: LookupMap::new(Key::Unclaimed),
        }
    }

    /// Upgrade hook: the new code is deployed by `upgrade`, then this re-reads the old state.
    /// State layout changes must be handled here before the new version ships.
    ///
    /// 2026-09-27: dropped `claim_limits`/`reward_rate_x18` (LogX/staking/rewards removed
    /// entirely, Development.md §16.11) — Borsh is positional, so removing fields from the middle
    /// of the struct means the old bytes no longer decode directly into `Self`. Reads the old
    /// layout explicitly (`PreLogxRemoval`, below) and carries every other field across unchanged,
    /// preserving all existing subaccounts, balances, positions, orders and prices instead of
    /// losing them to a fresh redeploy. One-time: safe to simplify back to a plain
    /// `env::state_read()` (and delete `PreLogxRemoval`/`PreLogxRemovalClaimLimits`) once this has
    /// actually run once against the deployed contract.
    #[private]
    #[init(ignore_state)]
    pub fn migrate() -> Self {
        let old: PreLogxRemoval = env::state_read().unwrap_or_else(|| env::panic_str("no state"));
        Self {
            owner: old.owner,
            guardian: old.guardian,
            sequencer: old.sequencer,
            chain_id: old.chain_id,
            broker_id: old.broker_id,
            n_submissions: old.n_submissions,
            paused: old.paused,
            migrated: old.migrated,
            fee_table: old.fee_table,
            amm_subaccount: old.amm_subaccount,
            insurance_subaccount: old.insurance_subaccount,
            fee_subaccount: old.fee_subaccount,
            price_max_age_sec: old.price_max_age_sec,
            max_session_ttl_ms: old.max_session_ttl_ms,
            min_new_deposit_x18: old.min_new_deposit_x18,
            max_order_ttl_ms: old.max_order_ttl_ms,
            spot_order: old.spot_order,
            collateral_order: old.collateral_order,
            spots: old.spots,
            perps: old.perps,
            token_to_spot: old.token_to_spot,
            subaccounts: old.subaccounts,
            owners: old.owners,
            session_keys: old.session_keys,
            nonces: old.nonces,
            filled: old.filled,
            product_accounts: old.product_accounts,
            side_products: old.side_products,
            options: old.options,
            pool_balances: old.pool_balances,
            trusted_depositors: old.trusted_depositors,
            held: old.held,
            unclaimed: old.unclaimed,
        }
    }

    // ------------------------------------------------------------------ batch entry point

    /// §5.4. One call = one receipt: any failing transaction panics and reverts the whole batch.
    /// Arguments are Borsh (the Go batcher encodes with near/borsh-go).
    pub fn submit_transactions(
        &mut self,
        #[serializer(borsh)] idx: u64,
        #[serializer(borsh)] txs: Vec<Vec<u8>>,
        #[serializer(borsh)] sigs: Vec<Vec<u8>>,
        #[serializer(borsh)] sigs2: Vec<Vec<u8>>,
    ) {
        self.assert_sequencer();
        if !self.migrated {
            env::panic_str("migration not finished");
        }
        if idx != self.n_submissions {
            env::panic_str(&format!("expected idx {}, got {idx}", self.n_submissions));
        }
        if txs.is_empty() || sigs.len() != txs.len() || sigs2.len() != txs.len() {
            env::panic_str("txs, sigs and sigs2 must have the same non-zero length");
        }
        let mut log = events::BatchLog::default();
        for (i, raw) in txs.iter().enumerate() {
            let tx_idx = idx + i as u64;
            let (ty, payload) = raw.split_first().unwrap_or_else(|| env::panic_str("empty transaction"));
            match *ty {
                tx::PERPTICK => self.tx_perp_tick(decode(payload), &mut log),
                tx::LIQUIDATE_SUBACCOUNT => self.tx_liquidate(decode(payload), &sigs[i], tx_idx, &mut log),
                tx::WITHDRAW_COLLATERAL => self.tx_withdraw(decode(payload), &sigs[i], tx_idx),
                tx::MATCH_ORDERS => self.tx_match(decode(payload), &sigs[i], &sigs2[i], tx_idx, &mut log),
                tx::SETTLE_USER_PNL => self.tx_settle_pnl(decode(payload), &mut log),
                tx::SOCIALISE_SUBACCOUNT => self.tx_socialise(decode(payload), &mut log),
                tx::SET_NONCE => self.tx_set_nonce(decode(payload)),
                tx::PLACE_OPTIONS_BET => self.tx_place_option(decode(payload), &sigs[i]),
                tx::CLOSE_OPTIONS_BET => self.tx_close_option(decode(payload), &mut log),
                tx::PRE_MARKET_ORDER_REQUEST => self.tx_pool_trade(side::PRE_MARKET, decode(payload), &sigs[i]),
                tx::SYN_SPOT_ORDER_REQUEST => self.tx_pool_trade(side::SYNTHETIC_SPOT, decode(payload), &sigs[i]),
                other => env::panic_str(&format!("unsupported transaction type {other}")),
            }
        }
        self.n_submissions = idx + txs.len() as u64;
        log.emit(idx, txs.len());
    }

    // ------------------------------------------------------------------ user entry points

    /// §6.3. The caller's NEAR account proves ownership by calling; the attached deposit pays for
    /// any new storage and the rest is refunded.
    #[payable]
    pub fn register_session_key(&mut self, subaccount_number: u64, session_key: String, expiry_ms: u64) {
        let owner = env::predecessor_account_id();
        let key = unhex20(&session_key);
        let now = now_ms();
        if expiry_ms <= now || expiry_ms > now + self.max_session_ttl_ms {
            env::panic_str("expiry must be in the future and within the maximum session length");
        }
        let sub = identity::subaccount_id(self.broker_id, owner.as_str(), subaccount_number);
        let before = env::storage_usage();
        self.bind_owner(&sub, &owner);
        self.session_keys.insert((sub, key), expiry_ms);
        self.session_keys.flush();
        self.owners.flush();
        self.charge_storage(before);
        emit("session_key_created", json!({ "subaccount": hex(&sub), "owner": owner, "session_key": hex(&key), "expiry": expiry_ms }));
    }

    /// Session keys for the system subaccounts (AMM, insurance, fee), which no NEAR wallet owns.
    /// Owner-only; the owner is the DAO, so this is the same trust as any other config change.
    pub fn register_system_session_key(&mut self, subaccount: String, session_key: String, expiry_ms: u64) {
        self.assert_owner();
        let sub = unhex32(&subaccount);
        if ![self.amm_subaccount, self.insurance_subaccount, self.fee_subaccount].contains(&sub) {
            env::panic_str("not a system subaccount");
        }
        if expiry_ms <= now_ms() {
            env::panic_str("expiry must be in the future");
        }
        let key = unhex20(&session_key);
        self.session_keys.insert((sub, key), expiry_ms);
        emit(
            "session_key_created",
            json!({ "subaccount": subaccount, "owner": self.owner, "session_key": hex(&key), "expiry": expiry_ms }),
        );
    }

    pub fn revoke_session_key(&mut self, subaccount_number: u64, session_key: String) {
        let owner = env::predecessor_account_id();
        let sub = identity::subaccount_id(self.broker_id, owner.as_str(), subaccount_number);
        self.session_keys.remove(&(sub, unhex20(&session_key)));
        emit("session_key_revoked", json!({ "subaccount": hex(&sub), "session_key": session_key }));
    }

    /// NEP-141 receiver (§7.1). msg is "" (credit the sender's subaccount 1) or JSON
    /// `{"account_id"?: "...", "subaccount_number"?: n, "source"?: "direct"|"1click"}`, or (owner only)
    /// `{"system": "amm"|"insurance"}` to fund a system subaccount.
    /// Returning the full amount refunds it (paused, unknown, or too small for a new subaccount).
    pub fn ft_on_transfer(&mut self, sender_id: AccountId, amount: U128, msg: String) -> PromiseOrValue<U128> {
        let token = env::predecessor_account_id();
        let pid = match self.token_to_spot.get(&token) {
            Some(p) => *p,
            None => env::panic_str("token not accepted"),
        };
        // S-6: how 1Click treats a refunded ft_transfer_call is unconfirmed, so deposits from a
        // trusted depositor (1Click's sender) are never refunded: they are credited, held while
        // deposits are closed, or parked as unclaimed if their msg cannot be read.
        let trusted = self.trusted_depositors.contains(&sender_id);
        // The AMM's capital and insurance top-ups: only the owner (DAO) may fund a system
        // subaccount, which no NEAR account owns and parse_deposit_msg cannot name.
        if let Some(target) = system_deposit_target(&msg) {
            if sender_id != self.owner {
                env::panic_str("only the owner funds system subaccounts");
            }
            if self.paused & pause::DEPOSITS != 0 || !self.migrated {
                env::panic_str("deposits are closed");
            }
            let sub_id = match target.as_str() {
                "amm" => self.amm_subaccount,
                "insurance" => self.insurance_subaccount,
                _ => env::panic_str("unknown system subaccount"),
            };
            let spot = self.spots.get(&pid).unwrap_or_else(|| env::panic_str("unknown spot")).clone();
            let scale = pow10(18u8.checked_sub(spot.decimals).unwrap_or_else(|| env::panic_str("decimals > 18")));
            let units = i128::try_from(amount.0).unwrap_or_else(|_| env::panic_str("amount too large"));
            let credit = units.checked_mul(scale).unwrap_or_else(|| env::panic_str("amount too large"));
            self.credit_quote_like(&sub_id, pid, credit);
            emit(
                "deposit",
                json!({ "subaccount": hex(&sub_id), "account_id": sender_id, "product_id": pid, "amount_x18": s(credit),
                        "sender_id": sender_id, "token": token, "amount": amount, "source": "system" }),
            );
            return PromiseOrValue::Value(U128(0));
        }
        let parsed = parse_deposit_msg(&msg, &sender_id);
        let (account, number, source) = match parsed {
            Ok(v) => v,
            Err(e) if trusted => {
                let units = amount.0;
                let prev = self.unclaimed.get(&pid).copied().unwrap_or(0);
                self.unclaimed.insert(pid, prev.checked_add(units).unwrap_or_else(|| env::panic_str("overflow")));
                emit(
                    "deposit_unclaimed",
                    json!({ "token": token, "product_id": pid, "sender_id": sender_id, "amount": amount, "msg": msg, "reason": e }),
                );
                return PromiseOrValue::Value(U128(0));
            }
            Err(e) => env::panic_str(e),
        };
        let spot = self.spots.get(&pid).unwrap_or_else(|| env::panic_str("unknown spot")).clone();
        let scale = pow10(18u8.checked_sub(spot.decimals).unwrap_or_else(|| env::panic_str("decimals > 18")));
        let units = i128::try_from(amount.0).unwrap_or_else(|_| env::panic_str("amount too large"));
        let credit = units.checked_mul(scale).unwrap_or_else(|| env::panic_str("amount too large"));
        if credit <= 0 {
            return PromiseOrValue::Value(amount);
        }
        let closed = self.paused & pause::DEPOSITS != 0 || !self.migrated;
        if closed {
            if !trusted {
                return PromiseOrValue::Value(amount);
            }
            let key = (account.clone(), number, pid);
            let prev = self.held.get(&key).copied().unwrap_or(0);
            self.held.insert(key, checked_add(prev, credit));
            emit(
                "deposit_held",
                json!({ "account_id": account, "subaccount_number": number, "product_id": pid, "amount_x18": s(credit), "sender_id": sender_id }),
            );
            return PromiseOrValue::Value(U128(0));
        }
        let sub_id = identity::subaccount_id(self.broker_id, account.as_str(), number);
        if !trusted && !self.subaccounts.contains_key(&sub_id) && credit < self.min_new_deposit_x18 {
            return PromiseOrValue::Value(amount);
        }
        self.credit_deposit(
            &account,
            number,
            pid,
            credit,
            json!({ "sender_id": sender_id, "token": token, "amount": amount, "source": source }),
        );
        PromiseOrValue::Value(U128(0))
    }

    /// Credits deposits held while deposits were closed. Anyone may call it once deposits are
    /// open: the funds can only go to the account named when they arrived.
    pub fn release_held_deposit(&mut self, account_id: AccountId, subaccount_number: u64, product_id: u32) -> U128 {
        if self.paused & pause::DEPOSITS != 0 || !self.migrated {
            env::panic_str("deposits are closed");
        }
        let key = (account_id.clone(), subaccount_number, product_id);
        let credit = self.held.remove(&key).unwrap_or_else(|| env::panic_str("nothing held"));
        self.credit_deposit(&account_id, subaccount_number, product_id, credit, json!({ "source": "held" }));
        U128(credit as u128)
    }

    /// DAO: assigns parked (unreadable-msg) deposits, in token units, to the account they belong to.
    pub fn assign_unclaimed(&mut self, product_id: u32, amount: U128, account_id: AccountId, subaccount_number: u64) {
        self.assert_owner();
        let have = self.unclaimed.get(&product_id).copied().unwrap_or(0);
        if amount.0 == 0 || amount.0 > have {
            env::panic_str("more than is unclaimed");
        }
        self.unclaimed.insert(product_id, have - amount.0);
        let spot = self.spots.get(&product_id).unwrap_or_else(|| env::panic_str("unknown spot")).clone();
        let credit = i128::try_from(amount.0)
            .ok()
            .and_then(|u| u.checked_mul(pow10(18 - spot.decimals)))
            .unwrap_or_else(|| env::panic_str("amount too large"));
        self.credit_deposit(&account_id, subaccount_number, product_id, credit, json!({ "source": "assigned" }));
    }

    /// Accounts whose deposits are never refunded (1Click's sender, e.g. intents.near).
    pub fn set_trusted_depositors(&mut self, accounts: Vec<AccountId>) {
        self.assert_owner();
        self.trusted_depositors = accounts;
    }

    pub fn get_held_deposit(&self, account_id: AccountId, subaccount_number: u64, product_id: u32) -> I128 {
        I128(self.held.get(&(account_id, subaccount_number, product_id)).copied().unwrap_or(0))
    }

    pub fn get_unclaimed(&self, product_id: u32) -> U128 {
        U128(self.unclaimed.get(&product_id).copied().unwrap_or(0))
    }

    /// §5.8 step 4. Never panics: a failed transfer re-credits the full debited amount.
    #[private]
    pub fn on_withdraw_complete(
        &mut self,
        subaccount: String,
        product_id: u32,
        amount_x18: U128,
        fee_x18: U128,
        payout: U128,
        receiver: AccountId,
        tx_idx: u64,
    ) -> bool {
        let sub_id = unhex32(&subaccount);
        // TooLong still means the transfer succeeded; only Failed re-credits
        let ok = !matches!(env::promise_result_checked(0, 64), Err(near_sdk::PromiseError::Failed));
        let data = json!({
            "subaccount": subaccount, "product_id": product_id, "amount": amount_x18, "fee": fee_x18,
            "payout": payout, "receiver": receiver, "tx_idx": tx_idx,
        });
        if ok {
            let fee_sub = self.fee_subaccount;
            self.credit_quote_like(&fee_sub, product_id, fee_x18.0 as i128);
            emit("withdraw_done", data);
        } else {
            self.credit_quote_like(&sub_id, product_id, amount_x18.0 as i128);
            emit("withdraw_failed", data);
        }
        ok
    }

    // ------------------------------------------------------------------ sequencer maintenance

    /// Removes replay-protection entries for orders that have expired (they can never fill again).
    pub fn prune_filled(&mut self, digests: Vec<String>) -> u32 {
        self.assert_sequencer();
        let now = now_ms();
        let mut removed = 0;
        for d in digests {
            let k = unhex32(&d);
            if matches!(self.filled.get(&k), Some(f) if f.expiration_ms < now) {
                self.filled.remove(&k);
                removed += 1;
            }
        }
        removed
    }

    // ------------------------------------------------------------------ admin (§5.10)

    pub fn pause(&mut self, mask: u8) {
        let caller = env::predecessor_account_id();
        if caller != self.guardian && caller != self.owner {
            env::panic_str("only guardian or owner");
        }
        self.paused |= mask & pause::ALL;
        emit("paused", json!({ "paused": self.paused, "by": caller }));
    }

    pub fn unpause(&mut self, mask: u8) {
        self.assert_owner();
        self.paused &= !mask;
        emit("paused", json!({ "paused": self.paused, "by": self.owner }));
    }

    pub fn set_roles(&mut self, owner: Option<AccountId>, guardian: Option<AccountId>, sequencer: Option<AccountId>) {
        self.assert_owner();
        if let Some(o) = owner {
            self.owner = o;
        }
        if let Some(g) = guardian {
            self.guardian = g;
        }
        if let Some(sq) = sequencer {
            self.sequencer = sq;
        }
    }

    /// D-7: the DAO moves accumulated fees out of the fee subaccount to a treasury account.
    /// Requires 1 yoctoNEAR (a deliberate, signed full-access action). The same debit, transfer
    /// and re-credit-on-failure path as user withdrawals; only whole token units are sent.
    #[payable]
    pub fn sweep_fees(&mut self, product_id: u32, amount_x18: U128, receiver: AccountId) -> Promise {
        self.assert_owner();
        near_sdk::assert_one_yocto();
        let spot = self.spots.get(&product_id).unwrap_or_else(|| env::panic_str("unknown spot")).clone();
        let token = spot.token.clone().unwrap_or_else(|| env::panic_str("product is not withdrawable"));
        let scale = pow10(18 - spot.decimals);
        let requested = i128::try_from(amount_x18.0).unwrap_or_else(|_| env::panic_str("amount too large"));
        let payout = requested / scale;
        let debit = payout * scale;
        if payout <= 0 {
            env::panic_str("amount too small");
        }
        let fee_id = self.fee_subaccount;
        let mut sub = self.load(&fee_id);
        if sub.spot(product_id) < debit {
            env::panic_str("fee balance too low");
        }
        sub.add_spot(product_id, checked_neg(debit));
        self.save(fee_id, sub);
        let sub_hex = hex(&fee_id);
        emit(
            "fee_sweep_pending",
            json!({ "product_id": product_id, "amount": s(debit), "payout": payout.to_string(), "receiver": receiver }),
        );
        Promise::new(token)
            .function_call(
                "ft_transfer".to_string(),
                json!({ "receiver_id": receiver, "amount": payout.to_string(), "memo": "near-stocks fee sweep" }).to_string().into_bytes(),
                ONE_YOCTO,
                GAS_FT_TRANSFER,
            )
            .then(Self::ext(env::current_account_id()).with_static_gas(GAS_WITHDRAW_CALLBACK).on_withdraw_complete(
                sub_hex,
                product_id,
                U128(debit as u128),
                U128(0),
                U128(payout as u128),
                receiver,
                self.n_submissions,
            ))
    }

    /// Counterparty subaccounts for options, pre-market and synthetic spot (0x-hex bytes32 each).
    pub fn set_product_accounts(
        &mut self,
        options_x: String,
        options_fees: String,
        pre_market_x: String,
        pre_market_fees: String,
        synthetic_x: String,
        synthetic_fees: String,
    ) {
        self.assert_owner();
        self.product_accounts = ProductAccounts {
            options_x: unhex32(&options_x),
            options_fees: unhex32(&options_fees),
            pre_market_x: unhex32(&pre_market_x),
            pre_market_fees: unhex32(&pre_market_fees),
            synthetic_x: unhex32(&synthetic_x),
            synthetic_fees: unhex32(&synthetic_fees),
        };
    }

    /// Enables a side product and sets its D-5 caps. kind: 0 options, 1 pre-market, 2 synthetic spot.
    pub fn set_side_product(&mut self, kind: u8, product_id: u32, config: SideProduct) {
        self.assert_owner();
        let ok = match kind {
            side::OPTIONS => config.max_fee <= 100 && config.max_payout_pct <= 1_000,
            side::PRE_MARKET | side::SYNTHETIC_SPOT => config.max_fee <= 10_000,
            _ => false,
        };
        if !ok {
            env::panic_str("invalid side product config");
        }
        self.side_products.insert((kind, product_id), config);
    }

    /// Mints a listed pre-market / synthetic product's supply to its house (x) account, the on-chain
    /// side of pre-markets.controller.go AddProduct (which credits PRE_MARKETS_X with MaxSupply).
    /// Owner-only, like listing the product with set_side_product.
    pub fn add_pool_supply(&mut self, kind: u8, product_id: u32, amount_x18: I128) {
        self.assert_owner();
        if amount_x18.0 <= 0 {
            env::panic_str("amount must be positive");
        }
        let x = match kind {
            side::PRE_MARKET => self.product_accounts.pre_market_x,
            side::SYNTHETIC_SPOT => self.product_accounts.synthetic_x,
            _ => env::panic_str("not a pool product"),
        };
        if x == [0u8; 32] {
            env::panic_str("product accounts not configured");
        }
        if !self.side_products.contains_key(&(kind, product_id)) {
            env::panic_str("product not listed");
        }
        self.add_pool_balance(kind, &x, product_id, amount_x18.0);
        emit("pool_supply", json!({ "kind": kind, "product_id": product_id, "subaccount": hex(&x), "amount_x18": s(amount_x18.0) }));
    }

    /// UPDATE_FEE_RATES (9) as an owner method (D-1).
    pub fn set_fee_table(&mut self, default_factor: I128, per_broker: Vec<(u64, I128)>) {
        self.assert_owner();
        let t = FeeTable { default_factor: default_factor.0, per_broker: per_broker.into_iter().map(|(b, f)| (b, f.0)).collect() };
        t.validate();
        self.fee_table = t;
    }

    /// UPDATE_PRODUCT (8) for spots. Keeps the stored price if `price_x18` is 0.
    pub fn upsert_spot(&mut self, product_id: u32, config: SpotConfig) {
        self.assert_owner();
        if !is_spot(product_id) {
            env::panic_str("spot ids are even");
        }
        let old = self.spots.get(&product_id).cloned();
        if let Some(t) = old.as_ref().and_then(|o| o.token.clone()) {
            self.token_to_spot.remove(&t);
        }
        if let Some(t) = &config.token {
            if matches!(self.token_to_spot.get(t), Some(p) if *p != product_id) {
                env::panic_str("token already mapped to another product");
            }
            self.token_to_spot.insert(t.clone(), product_id);
        }
        if config.decimals > 18 || config.withdraw_fee_x18.0 < 0 {
            env::panic_str("invalid spot config");
        }
        let price = if config.price_x18.0 > 0 {
            Price { price_x18: config.price_x18.0, updated_at: now_sec() }
        } else {
            old.map(|o| o.price).unwrap_or_default()
        };
        self.spots.insert(
            product_id,
            SpotProduct {
                token: config.token,
                decimals: config.decimals,
                weighted: config.weighted,
                withdraw_fee_x18: config.withdraw_fee_x18.0,
                price,
                max_deviation_bps: config.max_deviation_bps,
            },
        );
    }

    /// UPDATE_PRODUCT (8) for perps. Funding and open interest state are preserved.
    pub fn upsert_perp(&mut self, product_id: u32, config: PerpConfig) {
        self.assert_owner();
        if !is_perp(product_id) {
            env::panic_str("perp ids are odd");
        }
        let (imf, mmf, liq) = (config.imf_x18.0, config.mmf_x18.0, config.liq_frac_x18.0);
        if !(0 < mmf && mmf <= imf && imf <= E18 && (0..E18).contains(&liq)) || config.amm_max_position_x18.0 < 0 {
            env::panic_str("invalid margin config");
        }
        let mut p = self.perps.get(&product_id).cloned().unwrap_or(PerpProduct {
            imf_x18: 0,
            mmf_x18: 0,
            liq_frac_x18: 0,
            cum_funding_x18: 0,
            last_funding_time: 0,
            price: Price::default(),
            max_deviation_bps: 0,
            halted: false,
            open_interest_long: 0,
            open_interest_short: 0,
            amm_max_position_x18: 0,
        });
        p.imf_x18 = imf;
        p.mmf_x18 = mmf;
        p.liq_frac_x18 = liq;
        p.max_deviation_bps = config.max_deviation_bps;
        p.amm_max_position_x18 = config.amm_max_position_x18.0;
        if config.price_x18.0 > 0 {
            p.price = Price { price_x18: config.price_x18.0, updated_at: now_sec() };
        }
        self.perps.insert(product_id, p);
    }

    /// Owner override after a circuit breaker trip (§5.7): sets the price and clears the halt.
    pub fn set_price(&mut self, product_id: u32, price_x18: I128) {
        self.assert_owner();
        if price_x18.0 <= 0 {
            env::panic_str("price must be positive");
        }
        let price = Price { price_x18: price_x18.0, updated_at: now_sec() };
        if let Some(p) = self.perps.get_mut(&product_id) {
            p.price = price;
            p.halted = false;
        } else if let Some(sp) = self.spots.get_mut(&product_id) {
            sp.price = price;
        } else {
            env::panic_str("unknown product");
        }
        emit("price_set", json!({ "product_id": product_id, "price": price_x18 }));
    }

    pub fn set_halted(&mut self, product_id: u32, halted: bool) {
        let caller = env::predecessor_account_id();
        // guardian may only halt; clearing a halt is an owner decision
        if !(caller == self.owner || (caller == self.guardian && halted)) {
            env::panic_str("not allowed");
        }
        let p = self.perps.get_mut(&product_id).unwrap_or_else(|| env::panic_str("unknown perp"));
        p.halted = halted;
        emit("circuit_breaker", json!({ "product_id": product_id, "halted": halted, "by": caller }));
    }

    pub fn set_product_order(&mut self, spot_order: Vec<u32>, collateral_order: Vec<u32>) {
        self.assert_owner();
        for pid in spot_order.iter().chain(collateral_order.iter()) {
            if !self.spots.contains_key(pid) {
                env::panic_str("unknown spot in order list");
            }
        }
        self.spot_order = spot_order;
        self.collateral_order = collateral_order;
    }

    pub fn set_limits(&mut self, limits: Limits) {
        self.assert_owner();
        self.price_max_age_sec = limits.price_max_age_sec;
        self.max_session_ttl_ms = limits.max_session_ttl_ms;
        self.min_new_deposit_x18 = limits.min_new_deposit_x18.0;
        self.max_order_ttl_ms = limits.max_order_ttl_ms;
    }

    /// Owner-only code upgrade (the DAO enforces the timelock, §5.10). Input is the raw wasm.
    pub fn upgrade(&self) -> Promise {
        self.assert_owner();
        let code = env::input().unwrap_or_else(|| env::panic_str("missing code"));
        Promise::new(env::current_account_id()).deploy_contract(code).function_call(
            "migrate".to_string(),
            vec![],
            NearToken::from_yoctonear(0),
            Gas::from_tgas(20),
        )
    }

    /// §11: owner loads the snapshot in chunks while `migrated` is false.
    pub fn migrate_state(&mut self, chunk: Vec<MigrationEntry>, perps: Vec<PerpState>) {
        self.assert_owner();
        if self.migrated {
            env::panic_str("migration already finished");
        }
        for e in chunk {
            let id = unhex32(&e.subaccount);
            let mut sub = Subaccount::default();
            for (pid, b) in e.spots {
                sub.set_spot(pid, b.0);
            }
            for (pid, a, vq, last) in e.perps {
                sub.set_perp(pid, PerpBalance { amount: a.0, v_quote: vq.0, last_cum_funding: last.0 });
            }
            self.subaccounts.insert(id, sub);
            if let Some(o) = e.owner {
                self.owners.insert(id, o);
            }
            if e.nonce > 0 {
                self.nonces.insert(id, e.nonce);
            }
        }
        for ps in perps {
            let p = self.perps.get_mut(&ps.product_id).unwrap_or_else(|| env::panic_str("unknown perp"));
            p.cum_funding_x18 = ps.cum_funding_x18.0;
            p.last_funding_time = ps.last_funding_time;
            p.open_interest_long = ps.open_interest_long.0;
            p.open_interest_short = ps.open_interest_short.0;
        }
    }

    /// One-way latch (§5.10): after this, migrate_state is closed forever and batches may run.
    pub fn finish_migration(&mut self, n_submissions: u64) {
        self.assert_owner();
        if self.migrated {
            env::panic_str("migration already finished");
        }
        self.n_submissions = n_submissions;
        self.migrated = true;
        emit("migration_finished", json!({ "n_submissions": n_submissions }));
    }

    // ------------------------------------------------------------------ views

    pub fn n_submissions(&self) -> u64 {
        self.n_submissions
    }

    pub fn domain_separator(&self) -> String {
        hex(&self.domain())
    }

    pub fn get_config(&self) -> Value {
        json!({
            "owner": self.owner, "guardian": self.guardian, "sequencer": self.sequencer,
            "chain_id": self.chain_id, "broker_id": self.broker_id, "paused": self.paused,
            "migrated": self.migrated, "n_submissions": self.n_submissions,
            "fee_table": { "default_factor": s(self.fee_table.default_factor),
                "per_broker": self.fee_table.per_broker.iter().map(|(b, f)| json!([b, s(*f)])).collect::<Vec<_>>() },
            "amm_subaccount": hex(&self.amm_subaccount),
            "insurance_subaccount": hex(&self.insurance_subaccount),
            "fee_subaccount": hex(&self.fee_subaccount),
            "price_max_age_sec": self.price_max_age_sec, "max_session_ttl_ms": self.max_session_ttl_ms,
            "min_new_deposit_x18": s(self.min_new_deposit_x18), "max_order_ttl_ms": self.max_order_ttl_ms,
            "spot_order": self.spot_order, "collateral_order": self.collateral_order,
        })
    }

    pub fn get_products(&self) -> Value {
        let spots: Vec<Value> = self
            .spots
            .iter()
            .map(|(pid, p)| {
                json!({ "product_id": pid, "token": p.token, "decimals": p.decimals, "weighted": p.weighted,
                    "withdraw_fee_x18": s(p.withdraw_fee_x18), "price_x18": s(p.price.price_x18),
                    "price_updated_at": p.price.updated_at, "max_deviation_bps": p.max_deviation_bps })
            })
            .collect();
        let perps: Vec<Value> = self
            .perps
            .iter()
            .map(|(pid, p)| {
                json!({ "product_id": pid, "imf_x18": s(p.imf_x18), "mmf_x18": s(p.mmf_x18),
                    "liq_frac_x18": s(p.liq_frac_x18), "cum_funding_x18": s(p.cum_funding_x18),
                    "last_funding_time": p.last_funding_time, "price_x18": s(p.price.price_x18),
                    "price_updated_at": p.price.updated_at, "max_deviation_bps": p.max_deviation_bps,
                    "halted": p.halted, "amm_max_position_x18": s(p.amm_max_position_x18), "open_interest_long": s(p.open_interest_long),
                    "open_interest_short": s(p.open_interest_short) })
            })
            .collect();
        json!({ "spots": spots, "perps": perps })
    }

    pub fn get_subaccount(&self, subaccount: String) -> Value {
        let id = unhex32(&subaccount);
        let sub = self.subaccounts.get(&id).cloned().unwrap_or_default();
        json!({
            "subaccount": subaccount,
            "owner": self.owners.get(&id),
            "nonce": self.nonces.get(&id).copied().unwrap_or(0),
            "spots": sub.spots.iter().map(|(p, b)| json!([p, s(*b)])).collect::<Vec<_>>(),
            "perps": sub.perps.iter().map(|(p, b)| json!([p, s(b.amount), s(b.v_quote), s(b.last_cum_funding)])).collect::<Vec<_>>(),
        })
    }

    /// Health and withdrawable amounts using stored prices (the same numbers submit_transactions uses).
    pub fn get_health(&self, subaccount: String) -> Value {
        let id = unhex32(&subaccount);
        let sub = self.load(&id);
        let m = self.market(&[]);
        let h = risk::health(&sub, &m);
        let (mm, im) = risk::margins_x36(&sub, &m);
        let withdrawable: Vec<Value> =
            self.collateral_order.iter().map(|p| json!([p, s(risk::withdrawable_x18(&sub, &m, &self.collateral_order, *p))])).collect();
        json!({
            "equity_x36": risk::equity_x36(&sub, &m).to_string(),
            "initial_margin_x36": im.to_string(), "maintenance_margin_x36": mm.to_string(),
            "safety_x18": risk::safety_x18(&sub, &m).to_string(),
            "below_initial": h.below_initial, "below_maintenance": h.below_maintenance,
            "requires_insurance": h.requires_insurance, "withdrawable": withdrawable,
        })
    }

    pub fn get_option_bet(&self, order_id: u64) -> Value {
        self.options_view(order_id)
    }

    /// Pre-market (kind 1) or synthetic-spot (kind 2) token balance.
    pub fn get_pool_balance(&self, kind: u8, subaccount: String, product_id: u32) -> I128 {
        I128(self.pool_balance(kind, &unhex32(&subaccount), product_id))
    }

    pub fn get_side_products(&self) -> Value {
        json!(self.side_products.iter().map(|((k, p), c)| json!({ "kind": k, "product_id": p, "config": c })).collect::<Vec<_>>())
    }

    pub fn get_nonce(&self, subaccount: String) -> u64 {
        self.nonces.get(&unhex32(&subaccount)).copied().unwrap_or(0)
    }

    pub fn session_key_expiry(&self, subaccount: String, session_key: String) -> u64 {
        self.session_keys.get(&(unhex32(&subaccount), unhex20(&session_key))).copied().unwrap_or(0)
    }

    pub fn filled_amount(&self, digest: String) -> I128 {
        I128(self.filled.get(&unhex32(&digest)).map(|f| f.filled).unwrap_or(0))
    }

    pub fn subaccount_id_for(&self, account_id: AccountId, subaccount_number: u64) -> String {
        hex(&identity::subaccount_id(self.broker_id, account_id.as_str(), subaccount_number))
    }
}

// ---------------------------------------------------------------------- internals

/// The subaccount number NEAR users trade in: the backend's nearSubaccountNumber and the frontend's
/// `${broker}_${address}_1`.
pub const DEFAULT_SUBACCOUNT_NUMBER: u64 = 1;

fn parse_deposit_msg(msg: &str, sender: &AccountId) -> Result<(AccountId, u64, &'static str), &'static str> {
    if msg.trim().is_empty() {
        return Ok((sender.clone(), DEFAULT_SUBACCOUNT_NUMBER, "direct"));
    }
    let v: Value = near_sdk::serde_json::from_str(msg).map_err(|_| "invalid msg")?;
    let account = match v.get("account_id").and_then(|a| a.as_str()) {
        Some(a) => a.parse().map_err(|_| "invalid account_id")?,
        None => sender.clone(),
    };
    let number = v.get("subaccount_number").and_then(|n| n.as_u64()).unwrap_or(DEFAULT_SUBACCOUNT_NUMBER);
    let source = if v.get("source").and_then(|x| x.as_str()) == Some("1click") { "1click" } else { "direct" };
    Ok((account, number, source))
}

/// `{"system": "amm"|"insurance"}` names a system subaccount (owner-only deposits).
fn system_deposit_target(msg: &str) -> Option<String> {
    let v: Value = near_sdk::serde_json::from_str(msg).ok()?;
    v.get("system").and_then(|x| x.as_str()).map(|x| x.to_string())
}

fn sign(v: i128) -> i128 {
    v.signum()
}

impl NearStocks {
    fn assert_owner(&self) {
        if env::predecessor_account_id() != self.owner {
            env::panic_str("only owner");
        }
    }

    fn assert_sequencer(&self) {
        if env::predecessor_account_id() != self.sequencer {
            env::panic_str("only sequencer");
        }
    }

    fn assert_not_paused(&self, bit: u8) {
        if self.paused & bit != 0 {
            env::panic_str("paused");
        }
    }

    fn market<'a>(&'a self, overrides: &'a [(u32, i128)]) -> ChainMarket<'a> {
        ChainMarket { c: self, now_sec: now_sec(), overrides }
    }

    fn domain(&self) -> [u8; 32] {
        let vc = eip712::verifying_contract(env::current_account_id().as_str());
        eip712::domain_separator(self.chain_id, &vc)
    }

    fn load(&self, id: &[u8; 32]) -> Subaccount {
        self.subaccounts.get(id).cloned().unwrap_or_default()
    }

    fn save(&mut self, id: [u8; 32], sub: Subaccount) {
        self.subaccounts.insert(id, sub);
    }

    fn bind_owner(&mut self, sub: &[u8; 32], owner: &AccountId) {
        match self.owners.get(sub) {
            Some(o) if o != owner => env::panic_str("subaccount owned by another account"),
            Some(_) => {}
            None => {
                self.owners.insert(*sub, owner.clone());
            }
        }
    }

    fn charge_storage(&self, before: u64) {
        let used = env::storage_usage().saturating_sub(before);
        let cost = env::storage_byte_cost().saturating_mul(used as u128);
        let attached = env::attached_deposit();
        if attached < cost {
            env::panic_str(&format!("attach at least {} yoctoNEAR for storage", cost.as_yoctonear()));
        }
        let refund = attached.saturating_sub(cost);
        if !refund.is_zero() {
            Promise::new(env::predecessor_account_id()).transfer(refund).detach();
        }
    }

    /// Credits a deposit and emits the `deposit` event the indexer mirrors.
    fn credit_deposit(&mut self, account: &AccountId, number: u64, pid: u32, credit: i128, extra: Value) {
        let sub_id = identity::subaccount_id(self.broker_id, account.as_str(), number);
        self.bind_owner(&sub_id, account);
        let mut sub = self.load(&sub_id);
        sub.add_spot(pid, credit);
        self.save(sub_id, sub);
        let mut data = json!({ "subaccount": hex(&sub_id), "account_id": account, "product_id": pid, "amount_x18": s(credit) });
        if let (Some(d), Some(e)) = (data.as_object_mut(), extra.as_object()) {
            for (k, v) in e {
                d.insert(k.clone(), v.clone());
            }
        }
        emit("deposit", data);
    }

    fn credit_quote_like(&mut self, id: &[u8; 32], pid: u32, amount: i128) {
        if amount == 0 {
            return;
        }
        let mut sub = self.load(id);
        sub.add_spot(pid, amount);
        self.save(*id, sub);
    }

    /// §5.5 step 4: the signature must recover to `key`, and `key` must be live for `sub`.
    fn verify_session(&self, sub: &[u8; 32], key: &[u8; 20], digest: &[u8; 32], sig: &[u8]) {
        // user-specific refusals name the subaccount ("for 0x..."): the batcher pauses only it
        let signer = eip712::recover_signer(digest, sig).unwrap_or_else(|| env::panic_str(&format!("bad signature for {}", hex(sub))));
        if &signer != key {
            env::panic_str(&format!("signature does not match session key for {}", hex(sub)));
        }
        match self.session_keys.get(&(*sub, *key)) {
            Some(exp) if *exp > now_ms() => {}
            _ => env::panic_str(&format!("session key not registered or expired for {}", hex(sub))),
        }
    }

    /// Adds `fill` (signed like the order) to the order's filled amount; rejects overfill.
    fn record_fill(&mut self, digest: [u8; 32], order: &eip712::Order, fill: i128) -> i128 {
        if sign(fill) != sign(order.amount) {
            env::panic_str("fill direction does not match order");
        }
        let prev = self.filled.get(&digest).map(|f| f.filled).unwrap_or(0);
        let total = checked_add(prev, fill);
        if total.unsigned_abs() > order.amount.unsigned_abs() {
            env::panic_str("order overfilled");
        }
        self.filled.insert(digest, Fill { filled: total, expiration_ms: order.expiration });
        total
    }

    fn check_order(&self, o: &eip712::Order, pid: u32, sig: &[u8]) -> [u8; 32] {
        if o.product_id != pid {
            env::panic_str("order product mismatch");
        }
        let now = now_ms();
        if o.expiration <= now {
            env::panic_str("order expired");
        }
        if o.expiration > now + self.max_order_ttl_ms {
            env::panic_str("order expiration too far in the future");
        }
        let digest = eip712::typed_digest(&self.domain(), &eip712::order_struct_hash(o, self.chain_id));
        self.verify_session(&o.subaccount, &o.session_key, &digest, sig);
        digest
    }

    /// One side of a trade: position update, fee, reduce-only, circuit breaker, health, OI.
    /// Mirrors UpdateLocalBalanceForOrderMatch + DeductTradingFee + AddLongShortOIAtReddis.
    fn apply_trade(&mut self, id: &[u8; 32], pid: u32, d_a: i128, d_q: i128, is_reduce: bool) -> (i128, i128, i128) {
        let mut sub = self.load(id);
        let old = sub.perp(pid).amount;
        if is_reduce && !(old != 0 && sign(old) != sign(d_a) && d_a.unsigned_abs() <= old.unsigned_abs()) {
            env::panic_str("reduce-only order would increase position");
        }
        let perp = self.perps.get(&pid).unwrap_or_else(|| env::panic_str("unknown perp")).clone();
        let new_amount = checked_add(old, d_a);
        let increases = new_amount.unsigned_abs() > old.unsigned_abs() || sign(new_amount) * sign(old) < 0;
        if perp.halted && increases {
            env::panic_str("product halted: only reducing trades allowed");
        }
        let is_amm = *id == self.amm_subaccount;
        let (pnl, funding, fee);
        {
            let m = self.market(&[]);
            let before = if is_amm { I256::ZERO } else { risk::safety_x18(&sub, &m) };
            (pnl, funding) = sub.update_perp(pid, d_a, d_q, perp.cum_funding_x18);
            fee = if is_amm { 0 } else { perp::trading_fee(d_q, self.fee_table.factor(broker_of(id))) };
            sub.add_spot(QUOTE_PRODUCT_ID, checked_neg(fee));
            if !is_amm {
                let after = risk::safety_x18(&sub, &m);
                if !(after >= I256::ZERO || after >= before) {
                    env::panic_str(&format!("unhealthy trade for {}", hex(id)));
                }
            } else if !perp.amm_position_allowed(old, new_amount) {
                env::panic_str("AMM position cap");
            }
        }
        self.save(*id, sub);
        if !is_amm {
            self.perps.get_mut(&pid).unwrap().update_open_interest(old, d_a);
        }
        let fee_sub = self.fee_subaccount;
        self.credit_quote_like(&fee_sub, QUOTE_PRODUCT_ID, fee);
        (fee, pnl, funding)
    }

    // ---------------------------------------------------------------- transaction handlers

    fn tx_perp_tick(&mut self, t: tx::PerpTick, log: &mut events::BatchLog) {
        let now = now_sec();
        if t.time > now + 60 {
            env::panic_str("tick time in the future");
        }
        for (pid, rate) in t.rates {
            let p = self.perps.get_mut(&pid).unwrap_or_else(|| env::panic_str("unknown perp"));
            if p.last_funding_time != 0 {
                if t.time < p.last_funding_time {
                    env::panic_str("funding time went backwards");
                }
                let dt = (t.time - p.last_funding_time) as i128;
                p.cum_funding_x18 = checked_add(p.cum_funding_x18, rate.checked_mul(dt).unwrap_or_else(|| env::panic_str("i128 overflow")));
            }
            p.last_funding_time = t.time;
            log.funding.push(events::FundingLog { product_id: pid, cum_funding: p.cum_funding_x18, time: t.time });
        }
        for (pid, price) in t.prices {
            if price <= 0 {
                env::panic_str("price must be positive");
            }
            let (old, max_dev) = if is_perp(pid) {
                let p = self.perps.get(&pid).unwrap_or_else(|| env::panic_str("unknown perp"));
                (p.price, p.max_deviation_bps)
            } else {
                let p = self.spots.get(&pid).unwrap_or_else(|| env::panic_str("unknown spot"));
                (p.price, p.max_deviation_bps)
            };
            if !within_deviation(old, price, max_dev) {
                // circuit breaker: keep the old price, halt opening trades, alert
                if let Some(p) = self.perps.get_mut(&pid) {
                    p.halted = true;
                }
                emit(
                    "circuit_breaker",
                    json!({ "product_id": pid, "halted": true, "rejected_price": s(price), "last_price": s(old.price_x18) }),
                );
                continue;
            }
            let np = Price { price_x18: price, updated_at: now };
            if is_perp(pid) {
                self.perps.get_mut(&pid).unwrap().price = np;
            } else {
                self.spots.get_mut(&pid).unwrap().price = np;
            }
        }
    }

    fn tx_match(&mut self, p: tx::MatchOrders, taker_sig: &[u8], maker_sig: &[u8], tx_idx: u64, log: &mut events::BatchLog) {
        self.assert_not_paused(pause::TRADING);
        let pid = p.product_id;
        if !is_perp(pid) || !self.perps.contains_key(&pid) {
            env::panic_str("unknown perp");
        }
        let (t, mk) = (&p.taker, &p.maker);
        if t.subaccount == mk.subaccount {
            env::panic_str("self-trade");
        }
        if t.amount == 0 || sign(mk.amount) != -sign(t.amount) {
            env::panic_str("orders must be on opposite sides");
        }
        if p.matched_amount == 0 || sign(p.matched_amount) != sign(mk.amount) {
            env::panic_str("matched amount must carry the maker's sign");
        }
        if mk.price_x18 <= 0 || t.price_x18 < 0 {
            env::panic_str("invalid order price");
        }
        // price 0 = market order (order.controller.go); otherwise the limits must cross
        let taker_buys = t.amount > 0;
        if t.price_x18 != 0 && ((taker_buys && t.price_x18 < mk.price_x18) || (!taker_buys && t.price_x18 > mk.price_x18)) {
            env::panic_str("prices do not cross");
        }
        let td = self.check_order(t, pid, taker_sig);
        let md = self.check_order(mk, pid, maker_sig);
        self.record_fill(td, t, checked_neg(p.matched_amount));
        self.record_fill(md, mk, p.matched_amount);

        let d = perp::match_deltas(p.matched_amount.checked_abs().unwrap(), mk.price_x18, mk.amount > 0, taker_buys);
        // maker first, then taker (AtomicUpdateBalanceForOrderMatch)
        let (maker_fee, maker_pnl, maker_funding) = self.apply_trade(&mk.subaccount, pid, d.maker_a, d.maker_q, mk.is_reduce);
        let (taker_fee, taker_pnl, taker_funding) = self.apply_trade(&t.subaccount, pid, d.taker_a, d.taker_q, t.is_reduce);
        log.fills.push(events::FillLog { tx_idx, maker_fee, maker_pnl, maker_funding, taker_fee, taker_pnl, taker_funding });
    }

    fn tx_liquidate(&mut self, mut p: tx::Liquidate, liquidator_sig: &[u8], tx_idx: u64, log: &mut events::BatchLog) {
        self.assert_not_paused(pause::LIQUIDATIONS);
        let pid = p.product_id;
        let perp = self.perps.get(&pid).unwrap_or_else(|| env::panic_str("unknown perp")).clone();
        let lq = &p.liquidator;
        if lq.subaccount == p.liquidatee {
            env::panic_str("self-liquidation");
        }
        if p.amount == 0 || sign(lq.amount) != -sign(p.amount) {
            env::panic_str("liquidator order must take the other side");
        }
        let price = lq.price_x18;
        if price <= 0 {
            env::panic_str("invalid liquidator price");
        }
        p.prices = self.known_prices(&p.prices);
        let digest = self.check_order(lq, pid, liquidator_sig);
        self.record_fill(digest, lq, checked_neg(p.amount));

        // liquidatee must be below maintenance at the supplied prices, and may only be reduced
        let mut victim = self.load(&p.liquidatee);
        {
            let m = self.market(&p.prices);
            if !risk::health(&victim, &m).below_maintenance {
                env::panic_str("subaccount is not liquidatable");
            }
        }
        let old = victim.perp(pid).amount;
        if !(old != 0 && sign(old) != sign(p.amount) && p.amount.unsigned_abs() <= old.unsigned_abs()) {
            env::panic_str("liquidation must reduce the position");
        }

        // SettlePnlForLiquidatee
        let notional = mul_divx18(p.amount, price);
        let mut pb = victim.perp(pid);
        let (realised, funding) = pb.update(p.amount, checked_neg(notional), perp.cum_funding_x18);
        victim.set_perp(pid, pb);
        let abs_notional = notional.checked_abs().unwrap_or_else(|| env::panic_str("i128 overflow"));
        let liq_fee = mul_divx18(perp.liq_frac_x18, abs_notional);
        let trade_fee = perp::trading_fee(notional, self.fee_table.factor(broker_of(&p.liquidatee)));
        let quote = victim.spot(QUOTE_PRODUCT_ID);
        victim.add_spot(QUOTE_PRODUCT_ID, checked_neg(quote));
        let pnl_x36 = (wide(realised) - wide(liq_fee) - wide(trade_fee)) * wide(E18) + wide(quote) * wide(E18);
        {
            let m = self.market(&p.prices);
            risk::settle_using_spots(&mut victim, pnl_x36, &self.spot_order, &|x| m.spot_weighted(x), &|x| m.spot_price(x));
        }
        self.save(p.liquidatee, victim);
        if p.liquidatee != self.amm_subaccount {
            self.perps.get_mut(&pid).unwrap().update_open_interest(old, p.amount);
        }

        // SettlePnlForLiquidator (no fee). A non-AMM liquidator must pass the match health rule at
        // the liquidation's prices, and the AMM must stay within its cap (behavior-spec R-5); the
        // backend's LiquidationService.checkLiquidator applies the same rules first.
        let l_amount = checked_neg(p.amount);
        let l_notional = mul_divx18(l_amount, price);
        let mut liq = self.load(&lq.subaccount);
        let l_old = liq.perp(pid).amount;
        let is_amm = lq.subaccount == self.amm_subaccount;
        let (l_pnl, l_fund);
        {
            let m = self.market(&p.prices);
            let before = if is_amm { I256::ZERO } else { risk::safety_x18(&liq, &m) };
            (l_pnl, l_fund) = liq.update_perp(pid, l_amount, checked_neg(l_notional), perp.cum_funding_x18);
            if !is_amm {
                let after = risk::safety_x18(&liq, &m);
                if !(after >= I256::ZERO || after >= before) {
                    env::panic_str("liquidator would be unhealthy");
                }
            } else if !perp.amm_position_allowed(l_old, checked_add(l_old, l_amount)) {
                env::panic_str("AMM position cap");
            }
        }
        self.save(lq.subaccount, liq);
        if !is_amm {
            self.perps.get_mut(&pid).unwrap().update_open_interest(l_old, l_amount);
        }

        // D-2: liquidation fee to insurance; D-7: trading fee to the fee subaccount
        let (ins, fee_sub) = (self.insurance_subaccount, self.fee_subaccount);
        self.credit_quote_like(&ins, QUOTE_PRODUCT_ID, liq_fee);
        self.credit_quote_like(&fee_sub, QUOTE_PRODUCT_ID, trade_fee);
        log.liquidations.push(events::LiquidationLog {
            tx_idx,
            liquidation_fee: liq_fee,
            trading_fee: trade_fee,
            liquidatee_pnl: realised,
            liquidatee_funding: funding,
            liquidator_pnl: l_pnl,
            liquidator_funding: l_fund,
        });
    }

    fn tx_settle_pnl(&mut self, mut p: tx::SettleUserPnl, log: &mut events::BatchLog) {
        p.spot_prices = self.known_prices(&p.spot_prices);
        for id in p.subaccounts {
            let mut sub = self.load(&id);
            let quote = sub.spot(QUOTE_PRODUCT_ID);
            if quote >= 0 {
                continue; // Go returns "nothing to settle" for this subaccount and moves on
            }
            sub.add_spot(QUOTE_PRODUCT_ID, checked_neg(quote));
            let after = {
                let m = self.market(&p.spot_prices);
                risk::settle_using_spots(&mut sub, wide(quote) * wide(E18), &self.spot_order, &|x| m.spot_weighted(x), &|x| {
                    p.spot_prices.iter().find(|(q, _)| *q == x).map(|(_, v)| *v)
                })
            };
            self.save(id, sub);
            log.settled.push(events::SettleLog { subaccount: id, quote_after: after, insurance_paid: 0 });
        }
    }

    fn tx_socialise(&mut self, mut p: tx::Socialise, log: &mut events::BatchLog) {
        p.spot_prices = self.known_prices(&p.spot_prices);
        let id = p.subaccount;
        if id == self.insurance_subaccount {
            env::panic_str("cannot socialise the insurance fund");
        }
        let mut sub = self.load(&id);
        if sub.has_open_perps() {
            env::panic_str("subaccount still has open positions");
        }
        let quote = sub.spot(QUOTE_PRODUCT_ID);
        sub.add_spot(QUOTE_PRODUCT_ID, checked_neg(quote));
        let remaining = {
            let m = self.market(&p.spot_prices);
            risk::settle_using_spots(&mut sub, wide(quote) * wide(E18), &self.spot_order, &|x| m.spot_weighted(x), &|x| {
                p.spot_prices.iter().find(|(q, _)| *q == x).map(|(_, v)| *v)
            })
        };
        let mut paid = 0;
        if remaining < 0 {
            let ins_id = self.insurance_subaccount;
            let mut ins = self.load(&ins_id);
            ins.add_spot(QUOTE_PRODUCT_ID, remaining);
            if ins.spot(QUOTE_PRODUCT_ID) < 0 {
                env::panic_str("insurance is out of funds");
            }
            self.save(ins_id, ins);
            sub.add_spot(QUOTE_PRODUCT_ID, checked_neg(remaining));
            paid = checked_neg(remaining);
        }
        let quote_after = sub.spot(QUOTE_PRODUCT_ID);
        self.save(id, sub);
        log.settled.push(events::SettleLog { subaccount: id, quote_after, insurance_paid: paid });
    }

    fn tx_set_nonce(&mut self, p: tx::SetNonce) {
        if p.delta == 0 {
            env::panic_str("delta must be positive");
        }
        let n = self.nonces.get(&p.subaccount).copied().unwrap_or(0);
        self.nonces.insert(p.subaccount, n.checked_add(p.delta).unwrap_or_else(|| env::panic_str("nonce overflow")));
    }

    fn tx_withdraw(&mut self, w: eip712::NearWithdraw, sig: &[u8], tx_idx: u64) {
        self.assert_not_paused(pause::WITHDRAWALS);
        let pid = w.product_id;
        let spot = self.spots.get(&pid).unwrap_or_else(|| env::panic_str("unknown spot")).clone();
        let token = spot.token.clone().unwrap_or_else(|| env::panic_str("product is not withdrawable"));
        let receiver: AccountId = w.receiver.parse().unwrap_or_else(|_| env::panic_str("invalid receiver"));
        let digest = eip712::typed_digest(&self.domain(), &eip712::near_withdraw_struct_hash(&w, self.chain_id));
        self.verify_session(&w.subaccount, &w.session_key, &digest, sig);

        self.take_nonce(&w.subaccount, w.nonce);

        let amount = i128::try_from(w.amount).unwrap_or_else(|_| env::panic_str("amount too large"));
        let mut sub = self.load(&w.subaccount);
        // collateral: GetWithdrawableBalance; any other non-collateral spot token: the balance itself
        let allowed = if self.collateral_order.contains(&pid) {
            let m = self.market(&[]);
            risk::withdrawable_x18(&sub, &m, &self.collateral_order, pid)
        } else {
            sub.spot(pid).max(0)
        };
        if amount <= 0 || amount > allowed {
            env::panic_str("amount exceeds withdrawable balance");
        }
        if amount <= spot.withdraw_fee_x18 {
            env::panic_str("amount does not cover the withdrawal fee");
        }
        let scale = pow10(18 - spot.decimals);
        let net = amount - spot.withdraw_fee_x18;
        let payout = net / scale; // net > 0, so this equals Euclidean division
        if payout == 0 {
            env::panic_str("amount too small");
        }
        // fee plus sub-unit dust stays in the ledger (fee subaccount) so custody == ledger exactly
        let fee_total = amount - payout * scale;
        sub.add_spot(pid, checked_neg(amount));
        self.save(w.subaccount, sub);

        let sub_hex = hex(&w.subaccount);
        emit(
            "withdraw_pending",
            json!({ "subaccount": sub_hex, "product_id": pid, "amount": s(amount), "fee": s(fee_total),
                    "payout": payout.to_string(), "receiver": receiver, "tx_idx": tx_idx }),
        );
        Promise::new(token)
            .function_call(
                "ft_transfer".to_string(),
                json!({ "receiver_id": receiver, "amount": payout.to_string(), "memo": format!("near-stocks withdrawal {tx_idx}") })
                    .to_string()
                    .into_bytes(),
                ONE_YOCTO,
                GAS_FT_TRANSFER,
            )
            .then(Self::ext(env::current_account_id()).with_static_gas(GAS_WITHDRAW_CALLBACK).on_withdraw_complete(
                sub_hex,
                pid,
                U128(amount as u128),
                U128(fee_total as u128),
                U128(payout as u128),
                receiver,
                tx_idx,
            ))
            .detach();
    }

    /// §5.7: a sequencer-supplied price must stay within the product's deviation band of the
    /// stored price (an unset stored price or a 0 band accepts anything).
    /// The supplied prices for products this contract lists, each checked against its band. Prices
    /// for other products are dropped: the backend sends every collateral it knows (for example the
    /// legacy product 74), and a price for a product nobody here holds cannot matter.
    fn known_prices(&self, prices: &[(u32, i128)]) -> Vec<(u32, i128)> {
        prices
            .iter()
            .filter(|(pid, _)| if is_perp(*pid) { self.perps.contains_key(pid) } else { self.spots.contains_key(pid) })
            .inspect(|(pid, pr)| self.assert_near_stored(*pid, *pr))
            .copied()
            .collect()
    }

    fn assert_near_stored(&self, pid: u32, price: i128) {
        if price <= 0 {
            env::panic_str("price must be positive");
        }
        let (old, dev) = if is_perp(pid) {
            match self.perps.get(&pid) {
                Some(p) => (p.price, p.max_deviation_bps),
                None => env::panic_str("unknown perp"),
            }
        } else {
            match self.spots.get(&pid) {
                Some(p) => (p.price, p.max_deviation_bps),
                None => env::panic_str("unknown spot"),
            }
        };
        if !within_deviation(old, price, dev) {
            env::panic_str(&format!("price for product {pid} outside the allowed band"));
        }
    }
}

fn within_deviation(old: Price, new: i128, max_bps: u32) -> bool {
    if max_bps == 0 || old.updated_at == 0 || old.price_x18 <= 0 {
        return true;
    }
    let diff = wide(new) - wide(old.price_x18);
    diff.abs() * I256::from(10_000) <= wide(old.price_x18) * I256::from(max_bps)
}
