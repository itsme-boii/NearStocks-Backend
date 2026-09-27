//! Unit tests for options (24, 25), pre-market (26), synthetic spot (27), LogX staking and claims
//! (14, 15, 16, 18, 22) and LogX withdrawal (13). Numbers are worked by hand in the comments.
mod common;
use common::*;
use near_sdk::json_types::{I128, U128};
use near_sdk::NearToken;
use near_stocks_core::eip712::{self, ClaimLogX, ClaimRewards, NearWithdraw, OptionBet, PoolOrder, StakeRequest};
use near_stocks_core::events::hex;
use near_stocks_core::state::{side, ClaimLimits, SideProduct, LOGX_REWARDS_SUBACCOUNT};
use near_stocks_core::{tx, SpotConfig};

const LOGX: u32 = 0;
const STLOGX: u32 = 2;
const ALT: u32 = 1001; // a pre-market / synthetic product id
const E: i128 = E18;

fn alice() -> [u8; 32] {
    sub_of("alice.near", 0)
}

fn world() -> (Chain, Key) {
    let mut c = setup();
    c.call(owner(), |c| {
        c.set_product_accounts(hex(&options_x()), hex(&options_fees()), hex(&pre_x()), hex(&pre_fees()), hex(&syn_x()), hex(&syn_fees()));
        c.set_side_product(side::OPTIONS, BTC, SideProduct { enabled: true, max_fee: 10, max_payout_pct: 200 });
        c.set_side_product(side::PRE_MARKET, ALT, SideProduct { enabled: true, max_fee: 200, max_payout_pct: 0 });
        c.set_side_product(side::SYNTHETIC_SPOT, ALT, SideProduct { enabled: true, max_fee: 200, max_payout_pct: 0 });
        c.upsert_spot(
            LOGX,
            SpotConfig {
                token: Some(acc("logx.near")),
                decimals: 18,
                weighted: false,
                withdraw_fee_x18: I128(25 * E),
                price_x18: I128(0),
                max_deviation_bps: 0,
            },
        );
        c.upsert_spot(
            STLOGX,
            SpotConfig { token: None, decimals: 18, weighted: false, withdraw_fee_x18: I128(0), price_x18: I128(0), max_deviation_bps: 0 },
        );
        c.set_claim_limits(ClaimLimits { max_logx_claim_x18: I128(10_000 * E), max_reward_claim_x18: I128(500 * E) });
    });
    // the DAO funds the LogX rewards pool that claims are paid from
    fund_rewards(&mut c, 20_000 * E);
    let k = Key::new(1);
    register(&mut c, "alice.near", 0, &k);
    deposit(&mut c, "alice.near", 1_000);
    (c, k)
}

fn fund_rewards(c: &mut Chain, amount: i128) {
    let dao = owner();
    c.call(acc("logx.near"), |c| {
        let _ = c.ft_on_transfer(dao, U128(amount as u128), r#"{"system":"rewards"}"#.to_string());
    });
}

fn nonce(c: &Chain) -> u128 {
    c.view(|c| c.get_nonce(hex(&alice()))) as u128
}

// ------------------------------------------------------------------ options

fn bet(c: &Chain, k: &Key, amount: i128) -> OptionBet {
    OptionBet { subaccount: alice(), product_id: BTC, amount, interval: 5, nonce: nonce(c), session_key: k.addr }
}

fn place(c: &mut Chain, k: &Key, b: OptionBet, order_id: u64, entry: i128, payout: u32, fee: u32) {
    let sig = sign_struct(k, eip712::option_bet_hash(&b, CHAIN_ID));
    let p = tx::PlaceOption { bet: b, order_id, entry_price_x18: entry, payout_pct: payout, fee_pct: fee };
    Batch::new().push(common::env(tx::PLACE_OPTIONS_BET, &p), sig, vec![]).submit(c);
}

fn place_amt(c: &mut Chain, k: &Key, amount: i128, order_id: u64, entry: i128, payout: u32, fee: u32) {
    let b = bet(c, k, amount);
    place(c, k, b, order_id, entry, payout, fee);
}

fn close(c: &mut Chain, order_id: u64, exit: i128) {
    Batch::new().push(common::env(tx::CLOSE_OPTIONS_BET, &tx::CloseOption { order_id, exit_price_x18: exit }), vec![], vec![]).submit(c);
}

#[test]
fn options_win_pays_payout_minus_fee() {
    let (mut c, k) = world();
    let entry = 65_000 * E;
    // 0.002 BTC at 65,000: stake = Divx18(amount * entry) = 130 USDC
    place_amt(&mut c, &k, 2 * E / 1000, 7, entry, 180, 5);
    assert_eq!(spot(&c, &alice(), USDC), 870 * E);
    assert_eq!(spot(&c, &options_x(), USDC), 130 * E);
    assert_eq!(nonce(&c), 1);
    assert_eq!(c.view(|c| c.get_option_bet(7))["quote_delta"], (130 * E).to_string());
    // up bet, price rises: payout 234 (180%), fee 6.5 (5%), user +227.5
    close(&mut c, 7, 65_100 * E);
    let opts = events("batch")[0]["options"].clone();
    assert!(opts.is_string(), "close is recorded in the batch log");
    assert_eq!(spot(&c, &alice(), USDC), 10_975 * E / 10);
    assert_eq!(spot(&c, &options_x(), USDC), -104 * E);
    assert_eq!(spot(&c, &options_fees(), USDC), 65 * E / 10);
    assert!(c.view(|c| c.get_option_bet(7)).is_null(), "closed bets are removed");
    // closing again is a no-op
    close(&mut c, 7, 1);
    assert_eq!(spot(&c, &alice(), USDC), 10_975 * E / 10);
}

#[test]
fn options_loss_tie_and_down_bets() {
    let (mut c, k) = world();
    let entry = 65_000 * E;
    place_amt(&mut c, &k, 2 * E / 1000, 1, entry, 180, 5);
    close(&mut c, 1, entry); // tie = loss
    assert_eq!(spot(&c, &alice(), USDC), 870 * E);
    assert_eq!(spot(&c, &options_x(), USDC), 130 * E);
    // a down bet (negative amount): stake 65, wins when the price falls, payout 190% = 123.5
    place_amt(&mut c, &k, -E / 1000, 2, entry, 190, 0);
    assert_eq!(spot(&c, &alice(), USDC), 805 * E);
    close(&mut c, 2, 64_000 * E);
    assert_eq!(spot(&c, &alice(), USDC), 9_285 * E / 10);
    // unknown order ids are ignored
    close(&mut c, 999, entry);
}

#[test]
fn option_stake_rounds_like_go() {
    // Divx18 is Euclidean: -1 wei at 1.5 gives -2 wei (floor), +1 wei gives 1 wei
    let (mut c, k) = world();
    c.call(owner(), |c| c.set_side_product(side::OPTIONS, 9_001, SideProduct { enabled: true, max_fee: 10, max_payout_pct: 200 }));
    let before = spot(&c, &alice(), USDC);
    let mut b = bet(&c, &k, -1);
    b.product_id = 9_001;
    place(&mut c, &k, b, 50, 3 * E / 2, 100, 0);
    assert_eq!(before - spot(&c, &alice(), USDC), 2);
    let mut b = bet(&c, &k, 1);
    b.product_id = 9_001;
    place(&mut c, &k, b, 51, 3 * E / 2, 100, 0);
    assert_eq!(before - spot(&c, &alice(), USDC), 3);
    let mut b = bet(&c, &k, 1);
    b.product_id = 9_001;
    expect_panic("stake rounds to zero", || place(&mut c, &k, b, 52, E / 2, 100, 0));
}

#[test]
fn options_guards() {
    let (mut c, k) = world();
    let entry = 65_000 * E; // stakes below are base amounts: 0.002 BTC * 65,000 = 130 USDC
    expect_panic("insufficient margin", || place_amt(&mut c, &k, 2 * E / 100, 1, entry, 180, 5)); // 1,300 > 1,000
    expect_panic("outside product limits", || place_amt(&mut c, &k, E / 10_000, 1, entry, 250, 5));
    expect_panic("outside product limits", || place_amt(&mut c, &k, E / 10_000, 1, entry, 180, 11));
    expect_panic("outside the allowed band", || place_amt(&mut c, &k, E / 10_000, 1, 90_000 * E, 180, 5));
    let mut other = bet(&c, &k, E / 10_000);
    other.product_id = ETH;
    expect_panic("not enabled", || place(&mut c, &k, other, 1, 2_500 * E, 180, 5));
    let mut stale = bet(&c, &k, E / 10_000);
    stale.nonce = 5;
    expect_panic("bad nonce", || place(&mut c, &k, stale, 1, entry, 180, 5));
    place_amt(&mut c, &k, E / 10_000, 1, entry, 180, 5);
    expect_panic("already used", || place_amt(&mut c, &k, E / 10_000, 1, entry, 180, 5));
    // the signature covers the amount
    let b = bet(&c, &k, E / 10_000);
    let sig = sign_struct(&k, eip712::option_bet_hash(&b, CHAIN_ID));
    let mut forged = b.clone();
    forged.amount = 500 * E;
    let p = tx::PlaceOption { bet: forged, order_id: 2, entry_price_x18: entry, payout_pct: 180, fee_pct: 5 };
    expect_panic("signature does not match", || Batch::new().push(common::env(tx::PLACE_OPTIONS_BET, &p), sig, vec![]).submit(&mut c));
}

// ------------------------------------------------------------------ pre-market and synthetic spot

fn pool(c: &mut Chain, k: &Key, kind: u8, is_buy: bool, amount: i128, quote_delta: i128, fees: i128) {
    let o = PoolOrder { subaccount: alice(), product_id: ALT, amount, is_buy, nonce: nonce(c), session_key: k.addr };
    let ty = if kind == side::PRE_MARKET { eip712::PRE_MARKET_ORDER_TYPE } else { eip712::SYN_SPOT_ORDER_TYPE };
    let sig = sign_struct(k, eip712::pool_order_hash(ty, &o, CHAIN_ID));
    let code = if kind == side::PRE_MARKET { tx::PRE_MARKET_ORDER_REQUEST } else { tx::SYN_SPOT_ORDER_REQUEST };
    Batch::new().push(common::env(code, &tx::PoolTrade { order: o, quote_delta, fees }), sig, vec![]).submit(c);
}

fn pool_bal(c: &Chain, kind: u8, sub: &[u8; 32]) -> i128 {
    c.view(|c| c.get_pool_balance(kind, hex(sub), ALT)).0
}

#[test]
fn pre_market_buy_and_sell_move_the_right_ledgers() {
    let (mut c, k) = world();
    // buy: 100 gross, fee 1 (100 bps), 50 tokens out
    pool(&mut c, &k, side::PRE_MARKET, true, 100 * E, 50 * E, E);
    assert_eq!(spot(&c, &alice(), USDC), 900 * E);
    assert_eq!(spot(&c, &pre_x(), USDC), 99 * E);
    assert_eq!(spot(&c, &pre_fees(), USDC), E);
    assert_eq!(pool_bal(&c, side::PRE_MARKET, &alice()), 50 * E);
    assert_eq!(pool_bal(&c, side::PRE_MARKET, &pre_x()), -50 * E);
    // sell 20 tokens for 30 net, fee 0.3: the fee goes to the fees account's quote (H-4)
    pool(&mut c, &k, side::PRE_MARKET, false, 20 * E, 30 * E, 3 * E / 10);
    assert_eq!(spot(&c, &alice(), USDC), 930 * E);
    assert_eq!(pool_bal(&c, side::PRE_MARKET, &alice()), 30 * E);
    assert_eq!(pool_bal(&c, side::PRE_MARKET, &pre_x()), -30 * E);
    assert_eq!(spot(&c, &pre_fees(), USDC), E + 3 * E / 10);
    assert_eq!(spot(&c, &pre_x(), USDC), 99 * E - 30 * E - 3 * E / 10);
    // synthetic spot keeps its own ledger and accounts
    pool(&mut c, &k, side::SYNTHETIC_SPOT, true, 10 * E, 4 * E, 0);
    assert_eq!(pool_bal(&c, side::SYNTHETIC_SPOT, &alice()), 4 * E);
    assert_eq!(pool_bal(&c, side::PRE_MARKET, &alice()), 30 * E);
    assert_eq!(spot(&c, &syn_x(), USDC), 10 * E);
}

#[test]
fn pool_guards() {
    let (mut c, k) = world();
    expect_panic("fee above product cap", || pool(&mut c, &k, side::PRE_MARKET, true, 100 * E, 50 * E, 3 * E));
    expect_panic("insufficient margin", || pool(&mut c, &k, side::PRE_MARKET, true, 2_000 * E, 50 * E, 0));
    expect_panic("insufficient token balance", || pool(&mut c, &k, side::PRE_MARKET, false, E, E, 0));
    expect_panic("amounts must be positive", || pool(&mut c, &k, side::PRE_MARKET, true, 10 * E, 0, 0));
    c.call(owner(), |c| c.set_side_product(side::PRE_MARKET, ALT, SideProduct { enabled: false, max_fee: 200, max_payout_pct: 0 }));
    expect_panic("not enabled", || pool(&mut c, &k, side::PRE_MARKET, true, 10 * E, 5 * E, 0));
    expect_panic("invalid side product config", || {
        c.call(owner(), |c| c.set_side_product(side::PRE_MARKET, ALT, SideProduct { enabled: true, max_fee: 20_000, max_payout_pct: 0 }))
    });
}

// ------------------------------------------------------------------ LogX: claims, staking, withdrawal

fn claim_logx(c: &mut Chain, k: &Key, amount: i128) {
    let cl = ClaimLogX { subaccount: alice(), token_amount: amount, session_key: k.addr, nonce: nonce(c) };
    let sig = sign_struct(k, eip712::claim_logx_hash(&cl, CHAIN_ID));
    Batch::new().push(common::env(tx::CLAIM_LOGX, &cl), sig, vec![]).submit(c);
}

fn stake(c: &mut Chain, k: &Key, is_stake: bool, amount: i128) {
    let s =
        StakeRequest { subaccount: alice(), product_id: LOGX, amount, staker_contract: [0u8; 20], session_key: k.addr, nonce: nonce(c) };
    let ty = if is_stake { eip712::STAKE_TYPE } else { eip712::UNSTAKE_TYPE };
    let sig = sign_struct(k, eip712::stake_hash(ty, &s, CHAIN_ID));
    Batch::new().push(common::env(if is_stake { tx::STAKE_LOGX } else { tx::UNSTAKE_LOGX }, &s), sig, vec![]).submit(c);
}

#[test]
fn logx_claim_stake_unstake_and_rewards() {
    let (mut c, k) = world();
    claim_logx(&mut c, &k, 1_000 * E);
    assert_eq!(spot(&c, &alice(), LOGX), 1_000 * E);
    expect_panic("LogX claim outside limits", || claim_logx(&mut c, &k, 10_001 * E));
    stake(&mut c, &k, true, 400 * E);
    assert_eq!((spot(&c, &alice(), LOGX), spot(&c, &alice(), STLOGX)), (600 * E, 400 * E));
    expect_panic("insufficient token balance", || stake(&mut c, &k, true, 601 * E));
    stake(&mut c, &k, false, 100 * E);
    assert_eq!((spot(&c, &alice(), LOGX), spot(&c, &alice(), STLOGX)), (700 * E, 300 * E));
    expect_panic("insufficient token balance", || stake(&mut c, &k, false, 301 * E));

    let claim = |c: &mut Chain, amount: i128| {
        let r = ClaimRewards { subaccount: alice(), session_key: k.addr, staker_contract: [0u8; 20], product_id: LOGX, nonce: nonce(c) };
        let sig = sign_struct(&k, eip712::claim_rewards_hash(&r, CHAIN_ID));
        Batch::new().push(common::env(tx::CLAIM_REWARDS, &tx::ClaimRewardsTx { claim: r, amount_x18: amount }), sig, vec![]).submit(c);
    };
    claim(&mut c, 42 * E);
    assert_eq!(spot(&c, &alice(), LOGX), 742 * E);
    expect_panic("reward claim outside limits", || claim(&mut c, 501 * E));
    // claims are paid from the pool: 20,000 - 1,000 (airdrop) - 42 (rewards)
    assert_eq!(spot(&c, &LOGX_REWARDS_SUBACCOUNT, LOGX), 18_958 * E, "claimed LogX is backed, never minted");

    let tick = |c: &mut Chain, r: i128| {
        Batch::new().push(common::env(tx::REWARD_RATE_TICK, &tx::RewardRateTick { cumulative_rate_x18: r }), vec![], vec![]).submit(c)
    };
    tick(&mut c, 5 * E);
    assert_eq!(c.view(|c| c.get_reward_rate()).0, 5 * E);
    expect_panic("cannot decrease", || tick(&mut c, 4 * E));
}

#[test]
fn logx_withdraws_its_full_balance_through_the_near_withdrawal() {
    let (mut c, k) = world();
    claim_logx(&mut c, &k, 100 * E);
    let w = |c: &Chain, amount: i128| NearWithdraw {
        subaccount: alice(),
        session_key: k.addr,
        product_id: LOGX,
        amount: amount as u128,
        nonce: nonce(c),
        receiver: "alice.near".into(),
    };
    let send = |c: &mut Chain, w: NearWithdraw| {
        let d = eip712::typed_digest(&domain(), &eip712::near_withdraw_struct_hash(&w, CHAIN_ID));
        Batch::new().push(common::env(tx::WITHDRAW_LOGX, &w), k.sign(&d), vec![]).submit(c);
    };
    let too_much = w(&c, 101 * E);
    expect_panic("exceeds withdrawable", || send(&mut c, too_much));
    // LogX is not collateral: the whole balance is withdrawable, minus the 25 LogX fee
    let all = w(&c, 100 * E);
    send(&mut c, all);
    assert_eq!(spot(&c, &alice(), LOGX), 0);
    assert_eq!(events("withdraw_pending")[0]["payout"], (75 * E).to_string());
    let receipts = near_sdk::test_utils::get_created_receipts();
    assert_eq!(receipts[0].receiver_id.as_str(), "logx.near");
    let _ = (U128(0), NearToken::from_yoctonear(0));
}

#[test]
fn listing_mints_the_pool_supply_to_the_house_account() {
    let (mut c, k) = world();
    // pre-markets.controller.go AddProduct credits PRE_MARKETS_X with MaxSupply; the DAO mirrors it
    c.call(owner(), |c| c.add_pool_supply(side::PRE_MARKET, ALT, I128(1_000 * E)));
    assert_eq!(pool_bal(&c, side::PRE_MARKET, &pre_x()), 1_000 * E);
    pool(&mut c, &k, side::PRE_MARKET, true, 100 * E, 50 * E, E);
    assert_eq!(pool_bal(&c, side::PRE_MARKET, &pre_x()), 950 * E, "the house hands out from its supply, like the backend");
    expect_panic("only owner", || c.call(acc("mallory.near"), |c| c.add_pool_supply(side::PRE_MARKET, ALT, I128(E))));
    expect_panic("not a pool product", || c.call(owner(), |c| c.add_pool_supply(side::OPTIONS, BTC, I128(E))));
    expect_panic("product not listed", || c.call(owner(), |c| c.add_pool_supply(side::PRE_MARKET, 999, I128(E))));
    expect_panic("amount must be positive", || c.call(owner(), |c| c.add_pool_supply(side::SYNTHETIC_SPOT, ALT, I128(0))));
}

#[test]
fn claims_stop_when_the_rewards_pool_is_empty() {
    let (mut c, k) = world();
    claim_logx(&mut c, &k, 10_000 * E);
    claim_logx(&mut c, &k, 10_000 * E);
    assert_eq!(spot(&c, &LOGX_REWARDS_SUBACCOUNT, LOGX), 0);
    expect_panic("LogX rewards pool exhausted", || claim_logx(&mut c, &k, E));
    assert_eq!(spot(&c, &alice(), LOGX), 20_000 * E, "the refused claim changed nothing");
    // only the owner funds it, and only with LogX
    expect_panic("only the owner funds system subaccounts", || {
        c.call(acc("logx.near"), |c| {
            let _ = c.ft_on_transfer(acc("mallory.near"), U128(E as u128), r#"{"system":"rewards"}"#.to_string());
        })
    });
    let dao = owner();
    expect_panic("the rewards pool holds LogX only", || {
        c.call(acc("usdc.near"), |c| {
            let _ = c.ft_on_transfer(dao, U128(1_000_000), r#"{"system":"rewards"}"#.to_string());
        })
    });
    fund_rewards(&mut c, 5 * E);
    claim_logx(&mut c, &k, 5 * E);
}
