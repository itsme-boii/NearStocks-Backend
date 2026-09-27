//! Writes vectors/events.json: real NEP-297 logs emitted by the contract for every event the Go
//! indexer (services/indexer-server/nearindexer) mirrors. The Go tests parse exactly these lines.
//! Regenerate: EVENTS_OUT=../vectors/events.json cargo test -p near-stocks-core --test events_vectors
//! Without EVENTS_OUT the test checks the committed file is current.
mod common;
use common::*;
use near_sdk::json_types::U128;
use near_sdk::{NearToken, PromiseResult};
use near_stocks_core::eip712::NearWithdraw;
use near_stocks_core::events::hex;
use near_stocks_core::tx;

fn logs_of<R>(f: impl FnOnce() -> R) -> Vec<String> {
    f();
    near_sdk::test_utils::get_logs().into_iter().filter(|l| l.starts_with("EVENT_JSON:")).collect()
}

#[test]
fn contract_events_for_the_indexer() {
    let mut c = setup();
    let k = Key::new(1);
    register(&mut c, "alice.near", 1, &k);
    let a = sub_of("alice.near", 1);
    let mut out = serde_json::Map::new();

    out.insert(
        "deposit".into(),
        logs_of(|| c.call(usdc_token(), |c| c.ft_on_transfer(acc("alice.near"), U128(250_000_000), String::new()))).into(),
    );

    let w = NearWithdraw {
        subaccount: a,
        session_key: k.addr,
        product_id: USDC,
        amount: 100_250_000_000_000_000_000,
        nonce: 0,
        receiver: "alice.near".into(),
    };
    let batch = Batch::new().withdraw(w, &k);
    out.insert("withdraw_pending".into(), logs_of(|| batch.submit(&mut c)).into());
    let ah = hex(&a);
    out.insert(
        "withdraw_done".into(),
        logs_of(|| {
            c.callback(PromiseResult::Successful(vec![]), |c| {
                c.on_withdraw_complete(
                    ah.clone(),
                    USDC,
                    U128(100_250_000_000_000_000_000),
                    U128(500_000_000_000_000_000),
                    U128(99_750_000),
                    acc("alice.near"),
                    1,
                )
            })
        })
        .into(),
    );
    out.insert(
        "withdraw_failed".into(),
        logs_of(|| {
            c.callback(PromiseResult::Failed, |c| {
                c.on_withdraw_complete(
                    ah.clone(),
                    USDC,
                    U128(10_000_000_000_000_000_000),
                    U128(500_000_000_000_000_000),
                    U128(9_500_000),
                    acc("alice.near"),
                    2,
                )
            })
        })
        .into(),
    );
    out.insert(
        "fee_sweep_pending".into(),
        logs_of(|| {
            c.call_with(owner(), NearToken::from_yoctonear(1), NOW_MS, |c| {
                c.sweep_fees(USDC, U128(200_000_000_000_000_000), acc("treasury.near")).detach()
            })
        })
        .into(),
    );
    out.insert(
        "batch".into(),
        logs_of(|| Batch::new().push(common::env(tx::SET_NONCE, &tx::SetNonce { subaccount: a, delta: 1 }), vec![], vec![]).submit(&mut c))
            .into(),
    );
    out.insert("circuit_breaker".into(), logs_of(|| c.call(guardian(), |c| c.set_halted(BTC, true))).into());

    let v = serde_json::json!({
        "spec": "real contract logs (core/tests/events_vectors.rs); parsed by services/indexer-server/nearindexer",
        "contract": contract_id(),
        "subaccount": hex(&a),
        "feeSubaccount": hex(&fee_sub()),
        "events": out,
    });
    let text = serde_json::to_string_pretty(&v).unwrap() + "\n";
    let path = format!("{}/../../vectors/events.json", env!("CARGO_MANIFEST_DIR"));
    match std::env::var("EVENTS_OUT") {
        Ok(p) => std::fs::write(p, &text).unwrap(),
        Err(_) => assert_eq!(std::fs::read_to_string(&path).unwrap_or_default(), text, "vectors/events.json is stale: regenerate it"),
    }
}
