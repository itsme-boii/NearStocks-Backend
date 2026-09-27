//! Options, pre-market, synthetic spot, LogX staking and claims (behavior-spec §3.9, §3.10; D-5,
//! D-6, D-9). Ported from options.controller.go, pre-markets.controller.go,
//! synthetic-spots.controller.go and token.controller.go. Every user action is signed by a session
//! key under the near-stocks domain and consumes the subaccount's sequential nonce.
use crate::events::{self, emit, hex};
use crate::fixed::*;
use crate::state::*;
use crate::{eip712, risk, tx, NearStocks};
use near_sdk::env;
use near_sdk::serde_json::json;

impl NearStocks {
    /// The subaccount's nonce must equal `nonce`; it then advances (account.service.go
    /// WithNonceRedisLock + IncrementNonce).
    pub(crate) fn take_nonce(&mut self, sub: &[u8; 32], nonce: u128) {
        let current = self.nonces.get(sub).copied().unwrap_or(0);
        if nonce != current as u128 {
            env::panic_str(&format!("bad nonce for {}", hex(sub)));
        }
        self.nonces.insert(*sub, current + 1);
    }

    fn signed(&mut self, sub: &[u8; 32], key: &[u8; 20], struct_hash: [u8; 32], sig: &[u8], nonce: u128) {
        let digest = eip712::typed_digest(&self.domain(), &struct_hash);
        self.verify_session(sub, key, &digest, sig);
        self.take_nonce(sub, nonce);
    }

    fn side_product(&self, kind: u8, pid: u32) -> SideProduct {
        match self.side_products.get(&(kind, pid)) {
            Some(p) if p.enabled => *p,
            _ => env::panic_str("product not enabled"),
        }
    }

    fn account(&self, id: [u8; 32]) -> [u8; 32] {
        if id == [0u8; 32] {
            env::panic_str("product accounts not configured");
        }
        id
    }

    /// GetAvailableMargin(sub) >= spend, at stored prices.
    fn require_available(&self, id: &[u8; 32], sub: &Subaccount, spend: i128) {
        let m = self.market(&[]);
        if risk::available_margin_x18(sub, &m) < spend {
            env::panic_str(&format!("insufficient margin for {}", hex(id)));
        }
    }

    // ------------------------------------------------------------------ options (24, 25)

    pub(crate) fn tx_place_option(&mut self, p: tx::PlaceOption, sig: &[u8]) {
        self.assert_not_paused(pause::TRADING);
        let b = &p.bet;
        let cfg = self.side_product(side::OPTIONS, b.product_id);
        if b.amount == 0 {
            env::panic_str("stake must be non-zero");
        }
        if p.payout_pct > cfg.max_payout_pct || p.fee_pct > cfg.max_fee || p.entry_price_x18 <= 0 {
            env::panic_str("option terms outside product limits");
        }
        if self.options.contains_key(&p.order_id) {
            env::panic_str("option order id already used");
        }
        if self.perps.contains_key(&b.product_id) || self.spots.contains_key(&b.product_id) {
            self.assert_near_stored(b.product_id, p.entry_price_x18);
        }
        self.signed(&b.subaccount, &b.session_key, eip712::option_bet_hash(b, self.chain_id), sig, b.nonce);

        // the signed amount is in base units; the stake is the quote value at entry, exactly as
        // options.controller.go: quoteDelta = Divx18(amount * entryPrice), user quote -= |quoteDelta|
        let quote_delta = narrow(divx18(wide(b.amount) * wide(p.entry_price_x18)));
        if quote_delta == 0 {
            env::panic_str("stake rounds to zero");
        }
        let stake = quote_delta.checked_abs().unwrap_or_else(|| env::panic_str("i128 overflow"));
        let mut user = self.load(&b.subaccount);
        self.require_available(&b.subaccount, &user, stake);
        user.add_spot(QUOTE_PRODUCT_ID, checked_neg(stake));
        self.save(b.subaccount, user);
        let x = self.account(self.product_accounts.options_x);
        self.credit_quote_like(&x, QUOTE_PRODUCT_ID, stake);
        self.options.insert(
            p.order_id,
            StoredBet {
                subaccount: b.subaccount,
                product_id: b.product_id,
                quote_delta,
                entry_price_x18: p.entry_price_x18,
                payout_pct: p.payout_pct,
                fee_pct: p.fee_pct,
            },
        );
    }

    pub(crate) fn tx_close_option(&mut self, c: tx::CloseOption, log: &mut events::BatchLog) {
        // closing an unknown or already-closed bet is a no-op (options.controller.go)
        let Some(bet) = self.options.get(&c.order_id).copied() else { return };
        if c.exit_price_x18 <= 0 {
            env::panic_str("price must be positive");
        }
        if self.perps.contains_key(&bet.product_id) || self.spots.contains_key(&bet.product_id) {
            self.assert_near_stored(bet.product_id, c.exit_price_x18);
        }
        self.options.remove(&c.order_id);
        // a tie is a loss (strict comparison)
        let won = if bet.quote_delta > 0 { c.exit_price_x18 > bet.entry_price_x18 } else { c.exit_price_x18 < bet.entry_price_x18 };
        let (mut payout, mut fee) = (0, 0);
        if won {
            let stake = bet.quote_delta.unsigned_abs() as i128;
            payout = mul_div(stake, bet.payout_pct as i128, 100);
            fee = mul_div(stake, bet.fee_pct as i128, 100);
            let (x, fees) = (self.account(self.product_accounts.options_x), self.account(self.product_accounts.options_fees));
            self.credit_quote_like(&x, QUOTE_PRODUCT_ID, checked_neg(payout));
            self.credit_quote_like(&fees, QUOTE_PRODUCT_ID, fee);
            self.credit_quote_like(&bet.subaccount, QUOTE_PRODUCT_ID, checked_sub(payout, fee));
        }
        log.options.push(events::OptionCloseLog { order_id: c.order_id, won, payout, fee });
    }

    // ------------------------------------------------------------------ pre-market (26), synthetic spot (27)

    pub(crate) fn tx_pool_trade(&mut self, kind: u8, t: tx::PoolTrade, sig: &[u8]) {
        self.assert_not_paused(pause::TRADING);
        let o = &t.order;
        let cfg = self.side_product(kind, o.product_id);
        let (type_string, x, fees_acct) = match kind {
            side::PRE_MARKET => (eip712::PRE_MARKET_ORDER_TYPE, self.product_accounts.pre_market_x, self.product_accounts.pre_market_fees),
            _ => (eip712::SYN_SPOT_ORDER_TYPE, self.product_accounts.synthetic_x, self.product_accounts.synthetic_fees),
        };
        let (x, fees_acct) = (self.account(x), self.account(fees_acct));
        if o.amount <= 0 || t.quote_delta <= 0 || t.fees < 0 {
            env::panic_str("amounts must be positive");
        }
        // D-5: fee within the product's basis-point cap of the gross quote
        let gross = if o.is_buy { o.amount } else { checked_add(t.quote_delta, t.fees) };
        if wide(t.fees) * wide(10_000) > wide(gross) * wide(cfg.max_fee as i128) {
            env::panic_str("fee above product cap");
        }
        self.signed(&o.subaccount, &o.session_key, eip712::pool_order_hash(type_string, o, self.chain_id), sig, o.nonce);

        let pid = o.product_id;
        let mut user = self.load(&o.subaccount);
        if o.is_buy {
            // user pays the gross amount; the pool gets it net of the fee and hands out base tokens
            let net = checked_sub(o.amount, t.fees);
            if net < 0 {
                env::panic_str("fee exceeds amount");
            }
            self.require_available(&o.subaccount, &user, o.amount);
            user.add_spot(QUOTE_PRODUCT_ID, checked_neg(o.amount));
            self.save(o.subaccount, user);
            self.credit_quote_like(&x, QUOTE_PRODUCT_ID, net);
            self.credit_quote_like(&fees_acct, QUOTE_PRODUCT_ID, t.fees);
            self.add_pool_balance(kind, &x, pid, checked_neg(t.quote_delta));
            self.add_pool_balance(kind, &o.subaccount, pid, t.quote_delta);
        } else {
            // user sells base tokens; receives the net quote, the pool pays net plus the fee
            if self.pool_balance(kind, &o.subaccount, pid) < o.amount {
                env::panic_str(&format!("insufficient token balance for {}", hex(&o.subaccount)));
            }
            self.add_pool_balance(kind, &o.subaccount, pid, checked_neg(o.amount));
            self.add_pool_balance(kind, &x, pid, o.amount);
            // H-4 fixed: the sell fee goes to the fees account's quote balance
            self.credit_quote_like(&fees_acct, QUOTE_PRODUCT_ID, t.fees);
            self.credit_quote_like(&x, QUOTE_PRODUCT_ID, checked_neg(checked_add(t.quote_delta, t.fees)));
            user.add_spot(QUOTE_PRODUCT_ID, t.quote_delta);
            self.save(o.subaccount, user);
        }
    }

    pub(crate) fn pool_balance(&self, kind: u8, sub: &[u8; 32], pid: u32) -> i128 {
        self.pool_balances.get(&(kind, *sub, pid)).copied().unwrap_or(0)
    }

    pub(crate) fn add_pool_balance(&mut self, kind: u8, sub: &[u8; 32], pid: u32, delta: i128) {
        let v = checked_add(self.pool_balance(kind, sub, pid), delta);
        if v == 0 {
            self.pool_balances.remove(&(kind, *sub, pid));
        } else {
            self.pool_balances.insert((kind, *sub, pid), v);
        }
    }

    // ------------------------------------------------------------------ LogX staking and claims (D-9)

    /// STAKE_LOGX (15): LogX -> stLogX. UNSTAKE_LOGX (16): stLogX -> LogX. Same ledger move as
    /// token.controller.go; the unstake cooldown is enforced by the backend (behavior-spec R-7).
    pub(crate) fn tx_stake(&mut self, stake: bool, s: tx::Stake, sig: &[u8]) {
        let type_string = if stake { eip712::STAKE_TYPE } else { eip712::UNSTAKE_TYPE };
        if s.amount <= 0 {
            env::panic_str("amount must be positive");
        }
        self.signed(&s.subaccount, &s.session_key, eip712::stake_hash(type_string, &s, self.chain_id), sig, s.nonce);
        let (from, to) = if stake { (LOGX_PRODUCT_ID, STAKED_LOGX_PRODUCT_ID) } else { (STAKED_LOGX_PRODUCT_ID, LOGX_PRODUCT_ID) };
        let mut sub = self.load(&s.subaccount);
        if sub.spot(from) < s.amount {
            env::panic_str(&format!("insufficient token balance for {}", hex(&s.subaccount)));
        }
        sub.add_spot(from, checked_neg(s.amount));
        sub.add_spot(to, s.amount);
        self.save(s.subaccount, sub);
    }

    /// CLAIM_REWARDS (14): credits the backend-computed claimable LogX (D-6), capped per claim.
    pub(crate) fn tx_claim_rewards(&mut self, c: tx::ClaimRewardsTx, sig: &[u8]) {
        let cap = self.claim_limits.max_reward_claim_x18.0;
        if c.amount_x18 <= 0 || c.amount_x18 > cap {
            env::panic_str("reward claim outside limits");
        }
        let r = &c.claim;
        self.signed(&r.subaccount, &r.session_key, eip712::claim_rewards_hash(r, self.chain_id), sig, r.nonce);
        self.pay_from_rewards_pool(&r.subaccount, c.amount_x18);
    }

    /// Claims move LogX out of the DAO-funded rewards pool: a claim never creates LogX the contract
    /// does not hold, so LogX withdrawals stay fully backed.
    fn pay_from_rewards_pool(&mut self, to: &[u8; 32], amount: i128) {
        let mut pool = self.load(&LOGX_REWARDS_SUBACCOUNT);
        if pool.spot(LOGX_PRODUCT_ID) < amount {
            env::panic_str("LogX rewards pool exhausted: the DAO must top it up");
        }
        pool.add_spot(LOGX_PRODUCT_ID, checked_neg(amount));
        self.save(LOGX_REWARDS_SUBACCOUNT, pool);
        self.credit_quote_like(to, LOGX_PRODUCT_ID, amount);
    }

    /// CLAIM_LOGX (18): the airdrop amount the user signed and the backend verified, capped.
    pub(crate) fn tx_claim_logx(&mut self, c: tx::ClaimLogXTx, sig: &[u8]) {
        let cap = self.claim_limits.max_logx_claim_x18.0;
        if c.token_amount <= 0 || c.token_amount > cap {
            env::panic_str("LogX claim outside limits");
        }
        self.signed(&c.subaccount, &c.session_key, eip712::claim_logx_hash(&c, self.chain_id), sig, c.nonce);
        self.pay_from_rewards_pool(&c.subaccount, c.token_amount);
    }

    /// REWARD_RATE_TICK (22): recorded for audit; rewards themselves are computed off-chain (D-6).
    pub(crate) fn tx_reward_rate_tick(&mut self, t: tx::RewardRateTick) {
        if t.cumulative_rate_x18 < self.reward_rate_x18 {
            env::panic_str("reward rate cannot decrease");
        }
        self.reward_rate_x18 = t.cumulative_rate_x18;
        emit("reward_rate", json!({ "cumulative_rate_x18": t.cumulative_rate_x18.to_string(), "time": env::block_timestamp_ms() }));
    }

    pub(crate) fn options_view(&self, order_id: u64) -> near_sdk::serde_json::Value {
        match self.options.get(&order_id) {
            Some(b) => json!({
                "order_id": order_id, "subaccount": hex(&b.subaccount), "product_id": b.product_id,
                "quote_delta": b.quote_delta.to_string(), "entry_price_x18": b.entry_price_x18.to_string(),
                "payout_pct": b.payout_pct, "fee_pct": b.fee_pct,
            }),
            None => near_sdk::serde_json::Value::Null,
        }
    }
}
