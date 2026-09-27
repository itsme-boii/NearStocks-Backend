//! Shared fixture for the contract unit tests: a configured NearStocks in the mocked NEAR env,
//! secp256k1 session keys, and helpers that build signed batch transactions.
#![allow(dead_code)]
use near_sdk::borsh;
use near_sdk::json_types::{I128, U128};
use near_sdk::test_utils::VMContextBuilder;
use near_sdk::{AccountId, MockedBlockchain, NearToken, PromiseResult};
use near_stocks_core::eip712::{self, NearWithdraw, Order};
use near_stocks_core::events::hex;
use near_stocks_core::{identity, tx, NearStocks, PerpConfig, SpotConfig};
use secp256k1::{Message, Secp256k1, SecretKey};
use sha3::{Digest, Keccak256};
use std::collections::HashMap;

pub const E18: i128 = 1_000_000_000_000_000_000;
pub const CHAIN_ID: u64 = 397;
pub const BROKER: u64 = 2;
pub const NOW_MS: u64 = 1_760_000_000_000;
pub const EXPIRY: u64 = NOW_MS + 7 * 24 * 3600 * 1000;
pub const BTC: u32 = 1;
pub const ETH: u32 = 3;
pub const USDC: u32 = 4;

pub fn acc(s: &str) -> AccountId {
    s.parse().unwrap()
}

pub fn contract_id() -> AccountId {
    acc("near-stocks.near")
}
pub fn owner() -> AccountId {
    acc("dao.near")
}
pub fn guardian() -> AccountId {
    acc("guardian.near")
}
pub fn sequencer() -> AccountId {
    acc("sequencer.near")
}
pub fn usdc_token() -> AccountId {
    acc("usdc.near")
}

/// AMM_SUBACCOUNT_ID and INSURANCE_SUBACCOUNT_ID layouts from constants.utils.go; the fee
/// subaccount is new (D-7).
pub fn amm_sub() -> [u8; 32] {
    let mut b = [0u8; 32];
    b[5] = 1;
    b[25] = 1;
    b[31] = 1;
    b
}
pub fn insurance_sub() -> [u8; 32] {
    let mut b = [0u8; 32];
    b[25] = 5;
    b[31] = 1;
    b
}
pub fn fee_sub() -> [u8; 32] {
    let mut b = [0u8; 32];
    b[25] = 7;
    b[31] = 1;
    b
}

pub fn keccak(b: &[u8]) -> [u8; 32] {
    Keccak256::digest(b).into()
}

pub struct Key {
    pub sk: SecretKey,
    pub addr: [u8; 20],
}

impl Key {
    pub fn new(seed: u8) -> Self {
        let mut raw = [0u8; 32];
        raw[31] = seed;
        raw[0] = 0x11;
        let sk = SecretKey::from_slice(&raw).unwrap();
        let pk = sk.public_key(&Secp256k1::new()).serialize_uncompressed();
        let h = keccak(&pk[1..]);
        Key { sk, addr: h[12..].try_into().unwrap() }
    }

    pub fn sign(&self, digest: &[u8; 32]) -> Vec<u8> {
        let sig = Secp256k1::new().sign_ecdsa_recoverable(&Message::from_slice(digest).unwrap(), &self.sk);
        let (rid, bytes) = sig.serialize_compact();
        let mut out = bytes.to_vec();
        out.push(27 + rid.to_i32() as u8);
        out
    }

    pub fn hex(&self) -> String {
        hex(&self.addr)
    }
}

pub fn domain() -> [u8; 32] {
    eip712::domain_separator(CHAIN_ID, &eip712::verifying_contract(contract_id().as_str()))
}

pub fn order_digest(o: &Order) -> [u8; 32] {
    eip712::typed_digest(&domain(), &eip712::order_struct_hash(o, CHAIN_ID))
}

pub fn sub_of(account: &str, n: u64) -> [u8; 32] {
    identity::subaccount_id(BROKER, account, n)
}

fn context(predecessor: AccountId, deposit: NearToken, now_ms: u64) -> near_sdk::VMContext {
    let mut b = VMContextBuilder::new();
    b.current_account_id(contract_id())
        .predecessor_account_id(predecessor.clone())
        .signer_account_id(predecessor)
        .attached_deposit(deposit)
        .account_balance(NearToken::from_near(1000))
        .block_timestamp(now_ms * 1_000_000);
    b.build()
}

fn take_storage() -> HashMap<Vec<u8>, Vec<u8>> {
    near_sdk::mock::with_mocked_blockchain(|b| b.take_storage())
}

fn install(ctx: near_sdk::VMContext, storage: HashMap<Vec<u8>, Vec<u8>>, promise_results: Vec<PromiseResult>) {
    near_sdk::env::set_blockchain_interface(MockedBlockchain::new(
        ctx,
        near_sdk::test_vm_config(),
        near_sdk::RuntimeFeesConfig::test(),
        promise_results,
        storage,
        Default::default(),
        None,
    ));
}

/// Runs `f` like a NEAR function-call receipt: state is read from storage, written back on
/// success, and left exactly as it was if `f` panics (the whole receipt reverts).
fn exec<R>(ctx: near_sdk::VMContext, promise_results: Vec<PromiseResult>, f: impl FnOnce(&mut NearStocks) -> R) -> R {
    let before = take_storage();
    install(ctx.clone(), before.clone(), promise_results);
    let r = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
        let mut c: NearStocks = near_sdk::env::state_read().expect("contract not initialised");
        let r = f(&mut c);
        near_sdk::env::state_write(&c);
        drop(c); // store collections flush their caches on drop
        r
    }));
    match r {
        Ok(v) => v,
        Err(e) => {
            install(ctx, before, vec![]);
            std::panic::resume_unwind(e)
        }
    }
}

/// Handle to the contract living in mocked storage.
pub struct Chain;

impl Chain {
    pub fn call<R>(&mut self, who: AccountId, f: impl FnOnce(&mut NearStocks) -> R) -> R {
        exec(context(who, NearToken::from_yoctonear(0), NOW_MS), vec![], f)
    }

    pub fn call_with<R>(&mut self, who: AccountId, deposit: NearToken, now_ms: u64, f: impl FnOnce(&mut NearStocks) -> R) -> R {
        exec(context(who, deposit, now_ms), vec![], f)
    }

    /// a private callback, with the given result of the promise it follows
    pub fn callback<R>(&mut self, result: PromiseResult, f: impl FnOnce(&mut NearStocks) -> R) -> R {
        exec(context(contract_id(), NearToken::from_yoctonear(0), NOW_MS), vec![result], f)
    }

    /// Reads state in the current environment, so logs of the last call stay readable.
    pub fn view<R>(&self, f: impl FnOnce(&NearStocks) -> R) -> R {
        let c: NearStocks = near_sdk::env::state_read().expect("contract not initialised");
        f(&c)
    }

    pub fn deploy(f: impl FnOnce() -> NearStocks) -> Chain {
        install(context(owner(), NearToken::from_yoctonear(0), NOW_MS), HashMap::new(), vec![]);
        let c = f();
        near_sdk::env::state_write(&c);
        drop(c);
        Chain
    }
}

pub fn new_contract() -> NearStocks {
    NearStocks::new(owner(), guardian(), sequencer(), CHAIN_ID, BROKER, hex(&amm_sub()), hex(&insurance_sub()), hex(&fee_sub()))
}

pub fn spot_usdc() -> SpotConfig {
    SpotConfig {
        token: Some(usdc_token()),
        decimals: 6,
        weighted: true,
        withdraw_fee_x18: I128(E18 / 2),
        price_x18: I128(E18),
        max_deviation_bps: 200,
    }
}

pub fn perp_cfg(price: i128) -> PerpConfig {
    PerpConfig {
        imf_x18: I128(E18 / 10),
        mmf_x18: I128(E18 / 20),
        liq_frac_x18: I128(15 * E18 / 1000),
        price_x18: I128(price),
        max_deviation_bps: 1000,
        amm_max_position_x18: I128(0),
    }
}

/// A live contract: USDC spot, BTC and ETH perps, migration finished.
pub fn setup() -> Chain {
    let mut c = Chain::deploy(new_contract);
    c.call(owner(), |c| {
        c.upsert_spot(USDC, spot_usdc());
        c.upsert_perp(BTC, perp_cfg(65_000 * E18));
        c.upsert_perp(ETH, perp_cfg(2_500 * E18));
        c.set_product_order(vec![USDC], vec![USDC]);
        c.finish_migration(0);
    });
    c
}

pub fn register(c: &mut Chain, account: &str, n: u64, key: &Key) {
    let hexkey = key.hex();
    c.call_with(acc(account), NearToken::from_millinear(100), NOW_MS, |c| c.register_session_key(n, hexkey, NOW_MS + 24 * 3600 * 1000));
}

/// Deposits `usdc` whole units through ft_on_transfer, as the USDC contract would call it.
pub fn deposit(c: &mut Chain, account: &str, usdc: u128) {
    let r = c.call(usdc_token(), |c| c.ft_on_transfer(acc(account), U128(usdc * 1_000_000), r#"{"subaccount_number":0}"#.to_string()));
    match r {
        near_sdk::PromiseOrValue::Value(v) => assert_eq!(v.0, 0, "deposit refunded"),
        _ => panic!("unexpected promise"),
    }
}

static ORDER_SEQ: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);

/// Each call gets a distinct expiration (as real orders do, via their timestamp), so two
/// otherwise identical orders have different digests.
pub fn order(sub: [u8; 32], key: &Key, pid: u32, price: i128, amount: i128, is_reduce: bool) -> Order {
    let seq = ORDER_SEQ.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
    Order { subaccount: sub, price_x18: price, amount, expiration: EXPIRY + seq, is_reduce, session_key: key.addr, product_id: pid }
}

pub fn env(ty: u8, payload: &impl borsh::BorshSerialize) -> Vec<u8> {
    let mut v = vec![ty];
    v.extend(borsh::to_vec(payload).unwrap());
    v
}

pub struct Batch {
    pub txs: Vec<Vec<u8>>,
    pub sigs: Vec<Vec<u8>>,
    pub sigs2: Vec<Vec<u8>>,
}

impl Batch {
    pub fn new() -> Self {
        Batch { txs: vec![], sigs: vec![], sigs2: vec![] }
    }

    pub fn push(mut self, tx: Vec<u8>, s1: Vec<u8>, s2: Vec<u8>) -> Self {
        self.txs.push(tx);
        self.sigs.push(s1);
        self.sigs2.push(s2);
        self
    }

    /// taker/maker orders signed by their keys; matched has the maker's sign
    pub fn matched(self, taker: Order, tk: &Key, maker: Order, mk: &Key, matched: i128) -> Self {
        let ts = tk.sign(&order_digest(&taker));
        let ms = mk.sign(&order_digest(&maker));
        let pid = taker.product_id;
        self.push(env(tx::MATCH_ORDERS, &tx::MatchOrders { product_id: pid, taker, maker, matched_amount: matched }), ts, ms)
    }

    pub fn tick(self, time: u64, rates: Vec<(u32, i128)>, prices: Vec<(u32, i128)>) -> Self {
        self.push(env(tx::PERPTICK, &tx::PerpTick { time, rates, prices }), vec![], vec![])
    }

    pub fn withdraw(self, w: NearWithdraw, key: &Key) -> Self {
        let d = eip712::typed_digest(&domain(), &eip712::near_withdraw_struct_hash(&w, CHAIN_ID));
        let sig = key.sign(&d);
        self.push(env(tx::WITHDRAW_COLLATERAL, &w), sig, vec![])
    }

    pub fn submit(self, c: &mut Chain) {
        self.submit_at(c, NOW_MS)
    }

    pub fn submit_at(self, c: &mut Chain, now_ms: u64) {
        c.call_with(sequencer(), NearToken::from_yoctonear(0), now_ms, |c| {
            let idx = c.n_submissions();
            c.submit_transactions(idx, self.txs, self.sigs, self.sigs2)
        })
    }
}

pub fn spot(c: &Chain, sub: &[u8; 32], pid: u32) -> i128 {
    let v = c.view(|c| c.get_subaccount(hex(sub)));
    v["spots"]
        .as_array()
        .unwrap()
        .iter()
        .find(|e| e[0].as_u64().unwrap() as u32 == pid)
        .map(|e| e[1].as_str().unwrap().parse().unwrap())
        .unwrap_or(0)
}

/// (amount, v_quote, last_cum_funding)
pub fn perp(c: &Chain, sub: &[u8; 32], pid: u32) -> (i128, i128, i128) {
    let v = c.view(|c| c.get_subaccount(hex(sub)));
    v["perps"]
        .as_array()
        .unwrap()
        .iter()
        .find(|e| e[0].as_u64().unwrap() as u32 == pid)
        .map(|e| {
            let p = |i: usize| e[i].as_str().unwrap().parse::<i128>().unwrap();
            (p(1), p(2), p(3))
        })
        .unwrap_or((0, 0, 0))
}

pub fn logs() -> Vec<String> {
    near_sdk::test_utils::get_logs()
}

pub fn events(name: &str) -> Vec<serde_json::Value> {
    logs()
        .iter()
        .filter_map(|l| l.strip_prefix("EVENT_JSON:"))
        .map(|j| serde_json::from_str::<serde_json::Value>(j).unwrap())
        .filter(|v| v["event"] == name)
        .map(|v| v["data"][0].clone())
        .collect()
}

/// Events emitted by the most recent call.
/// Asserts that `f` panics with a message containing `needle`.
pub fn expect_panic<F: FnOnce()>(needle: &str, f: F) {
    let r = std::panic::catch_unwind(std::panic::AssertUnwindSafe(f));
    let err = r.err().unwrap_or_else(|| panic!("expected panic containing {needle:?}"));
    let msg = err.downcast_ref::<String>().cloned().or_else(|| err.downcast_ref::<&str>().map(|s| s.to_string())).unwrap_or_default();
    assert!(msg.contains(needle), "panic {msg:?} does not contain {needle:?}");
}

/// Digests for a contract deployed at an arbitrary account (sandbox tests).
pub fn domain_in(contract: &str) -> [u8; 32] {
    eip712::domain_separator(CHAIN_ID, &eip712::verifying_contract(contract))
}

pub fn order_digest_in(contract: &str, o: &Order) -> [u8; 32] {
    eip712::typed_digest(&domain_in(contract), &eip712::order_struct_hash(o, CHAIN_ID))
}

pub fn withdraw_digest_in(contract: &str, w: &NearWithdraw) -> [u8; 32] {
    eip712::typed_digest(&domain_in(contract), &eip712::near_withdraw_struct_hash(w, CHAIN_ID))
}

/// Decodes the packed records of the last call's `batch` event.
pub fn batch_log() -> (
    Vec<near_stocks_core::events::FillLog>,
    Vec<near_stocks_core::events::LiquidationLog>,
    Vec<near_stocks_core::events::FundingLog>,
    Vec<near_stocks_core::events::SettleLog>,
) {
    let ev = events("batch");
    let b = &ev[0];
    fn dec<T: borsh::BorshDeserialize>(v: &serde_json::Value) -> Vec<T> {
        match v.as_str() {
            Some(s) => {
                let bytes: near_sdk::json_types::Base64VecU8 = serde_json::from_value(serde_json::Value::String(s.into())).unwrap();
                borsh::from_slice(&bytes.0).unwrap()
            }
            None => vec![],
        }
    }
    (dec(&b["fills"]), dec(&b["liquidations"]), dec(&b["funding"]), dec(&b["settled"]))
}

/// Signs an EIP-712 struct hash under the near-stocks.near domain.
pub fn sign_struct(key: &Key, struct_hash: [u8; 32]) -> Vec<u8> {
    key.sign(&eip712::typed_digest(&domain(), &struct_hash))
}

pub fn options_x() -> [u8; 32] {
    let mut b = [0u8; 32];
    b[5] = 1;
    b[25] = 8;
    b[31] = 1;
    b
}
pub fn options_fees() -> [u8; 32] {
    let mut b = options_x();
    b[25] = 9;
    b
}
pub fn pre_x() -> [u8; 32] {
    let mut b = options_x();
    b[25] = 0x0b;
    b
}
pub fn pre_fees() -> [u8; 32] {
    let mut b = options_x();
    b[25] = 0x0c;
    b
}
pub fn syn_x() -> [u8; 32] {
    let mut b = options_x();
    b[25] = 0x0d;
    b
}
pub fn syn_fees() -> [u8; 32] {
    let mut b = options_x();
    b[25] = 0x0e;
    b
}
