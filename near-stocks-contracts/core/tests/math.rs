//! Math parity (behavior-spec §3.1-3.2, gate G3 core): Rust must equal the production Go code.
use near_stocks_core::perp::{match_deltas, trading_fee, FeeTable, PerpBalance};
use serde_json::Value;

fn load() -> Value {
    let path = format!("{}/../../vectors/math.json", env!("CARGO_MANIFEST_DIR"));
    serde_json::from_str(&std::fs::read_to_string(&path).expect(&path)).unwrap()
}
fn i(v: &Value) -> i128 {
    v.as_str().unwrap().parse().unwrap()
}
fn state(v: &Value) -> PerpBalance {
    PerpBalance { amount: i(&v["amount"]), v_quote: i(&v["vQuote"]), last_cum_funding: i(&v["lastCum"]) }
}

#[test]
fn update_balance_matches_go() {
    let v = load();
    let cases = v["updateBalance"].as_array().unwrap();
    assert!(cases.len() >= 700);
    for (n, c) in cases.iter().enumerate() {
        let mut pb = state(&c["before"]);
        let (pnl, fee) = pb.update(i(&c["deltaA"]), i(&c["deltaQ"]), i(&c["cumNow"]));
        assert_eq!(pb, state(&c["after"]), "case {n} state");
        assert_eq!(pnl, i(&c["pnl"]), "case {n} pnl");
        assert_eq!(fee, i(&c["fundingFee"]), "case {n} funding fee");
    }
}

#[test]
fn match_deltas_match_engine() {
    let v = load();
    for (n, c) in v["matchDeltas"].as_array().unwrap().iter().enumerate() {
        let d = match_deltas(i(&c["matched"]), i(&c["price"]), c["makerIsBuy"].as_bool().unwrap(), c["takerIsBuy"].as_bool().unwrap());
        assert_eq!(d.maker_a, i(&c["makerDeltaA"]), "case {n}");
        assert_eq!(d.maker_q, i(&c["makerDeltaQ"]), "case {n}");
        assert_eq!(d.taker_a, i(&c["takerDeltaA"]), "case {n}");
        assert_eq!(d.taker_q, i(&c["takerDeltaQ"]), "case {n}");
    }
}

#[test]
fn trading_fee_matches_go() {
    let v = load();
    let mut nonzero = 0;
    for (n, c) in v["tradingFee"].as_array().unwrap().iter().enumerate() {
        let fee = trading_fee(i(&c["deltaQ"]), FeeTable::legacy().factor(c["brokerId"].as_u64().unwrap()));
        assert_eq!(fee, i(&c["fee"]), "case {n}");
        if fee != 0 {
            nonzero += 1;
        }
    }
    assert!(nonzero > 0, "broker 2 cases must exercise a non-zero fee");
}

#[test]
fn euclidean_not_truncating() {
    // Guards the Go-parity rule directly: Divx18(-1) must be -1, not 0.
    assert_eq!(near_stocks_core::fixed::mul_divx18(-1, 1), -1);
    assert_eq!(near_stocks_core::fixed::mul_div(7, 1, -2), -3);
    assert_eq!(near_stocks_core::fixed::mul_div(-7, 1, 2), -4);
}

#[test]
fn fee_table_config() {
    let t = FeeTable { default_factor: 25, per_broker: vec![(2, 60), (7, 0)] };
    t.validate();
    assert_eq!(t.factor(2), 60);
    assert_eq!(t.factor(7), 0);
    assert_eq!(t.factor(1), 25);
    // 1000 quote at 0.025% = 0.25
    assert_eq!(trading_fee(-1_000 * near_stocks_core::fixed::E18, t.factor(1)), 250_000_000_000_000_000);
}

#[test]
#[should_panic(expected = "fee factor out of range")]
fn fee_table_rejects_excessive_fee() {
    FeeTable { default_factor: 1_001, per_broker: vec![] }.validate();
}
