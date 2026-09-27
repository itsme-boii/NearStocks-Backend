//! Gate G3 (core): replays vectors/parity.json, produced by the production Go ledger code
//! (services/balance-server/tests/parity_gen_test.go), through submit_transactions. Every
//! accepted step must be accepted, every rejected step must be rejected for the same reason, and
//! the final balances, fee subaccount, open interest and funding must match exactly.
mod common;
use common::*;
use near_sdk::json_types::I128;
use near_sdk::NearToken;
use near_stocks_core::eip712::{NearWithdraw, Order};
use near_stocks_core::events::{hex, unhex32};
use near_stocks_core::{tx, Limits, MigrationEntry, NearStocks, PerpConfig, SpotConfig};
use serde_json::Value;
use std::collections::HashMap;

fn load() -> Value {
    let p = format!("{}/../../vectors/parity.json", env!("CARGO_MANIFEST_DIR"));
    serde_json::from_str(&std::fs::read_to_string(&p).expect(&p)).unwrap()
}

fn int(v: &Value) -> i128 {
    v.as_str().map(|s| s.parse().unwrap()).unwrap_or_else(|| v.as_i64().unwrap() as i128)
}

fn pid(v: &Value) -> u32 {
    int(v) as u32
}

fn pairs(v: &Value) -> Vec<(u32, i128)> {
    v.as_array().map(|a| a.iter().map(|p| (pid(&p[0]), int(&p[1]))).collect()).unwrap_or_default()
}

struct Replay {
    c: Chain,
    keys: HashMap<[u8; 32], Key>,
    names: HashMap<[u8; 32], String>,
    nonces: HashMap<[u8; 32], u128>,
    now_ms: u64,
    spot_prices: Vec<(u32, i128)>,
    all_prices: Vec<(u32, i128)>,
}

impl Replay {
    fn key(&self, sub: &[u8; 32]) -> &Key {
        self.keys.get(sub).expect("no key for party")
    }

    fn order(&self, sub: [u8; 32], pid: u32, price: i128, amount: i128, reduce: bool) -> Order {
        order(sub, self.key(&sub), pid, price, amount, reduce)
    }

    /// Returns Err(panic message) when the batch reverts.
    fn submit(&mut self, b: Batch) -> Result<(), String> {
        let now = self.now_ms;
        let r = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| b.submit_at(&mut self.c, now)));
        r.map_err(|e| e.downcast_ref::<String>().cloned().unwrap_or_default())
    }
}

fn deploy(sc: &Value) -> Replay {
    let (amm, ins) = (sc["amm"].as_str().unwrap().to_string(), sc["insurance"].as_str().unwrap().to_string());
    let fee = sc["feeSub"].as_str().unwrap().to_string();
    let start_ms = sc["startSec"].as_u64().unwrap() * 1000;
    let mut c =
        Chain::deploy(|| NearStocks::new(owner(), guardian(), sequencer(), CHAIN_ID, BROKER, amm.clone(), ins.clone(), fee.clone()));
    let spots: Vec<u32> = sc["spots"].as_array().unwrap().iter().map(|v| v.as_u64().unwrap() as u32).collect();
    let entries: Vec<MigrationEntry> = sc["initial"]
        .as_array()
        .unwrap()
        .iter()
        .map(|st| MigrationEntry {
            subaccount: st["sub"].as_str().unwrap().to_string(),
            owner: None,
            nonce: 0,
            spots: st["spots"].as_array().unwrap().iter().map(|p| (pid(&p[0]), I128(int(&p[1])))).collect(),
            perps: st["perps"]
                .as_array()
                .unwrap()
                .iter()
                .map(|p| (pid(&p[0]), I128(int(&p[1])), I128(int(&p[2])), I128(int(&p[3]))))
                .collect(),
        })
        .collect();
    let perps = sc["perps"].as_array().unwrap().clone();
    c.call_with(owner(), NearToken::from_yoctonear(0), start_ms, |c| {
        for (i, s) in spots.iter().enumerate() {
            // no withdrawal fee so the debit equals Go's; 6 decimals like USDC
            let token = if i == 0 { usdc_token() } else { acc(&format!("spot{s}.near")) };
            c.upsert_spot(
                *s,
                SpotConfig {
                    token: Some(token),
                    decimals: 6,
                    weighted: true,
                    withdraw_fee_x18: I128(0),
                    price_x18: I128(0),
                    max_deviation_bps: 1000,
                },
            );
        }
        for p in &perps {
            c.upsert_perp(
                pid(&p[0]),
                PerpConfig {
                    imf_x18: I128(int(&p[1])),
                    mmf_x18: I128(int(&p[2])),
                    liq_frac_x18: I128(int(&p[3])),
                    price_x18: I128(0),
                    max_deviation_bps: 1000,
                    amm_max_position_x18: I128(int(&p[4])),
                },
            );
        }
        c.set_product_order(spots.clone(), spots.clone());
        c.set_limits(Limits {
            price_max_age_sec: 300,
            max_session_ttl_ms: 7 * 24 * 3600 * 1000,
            min_new_deposit_x18: I128(E18),
            max_order_ttl_ms: 31 * 24 * 3600 * 1000,
        });
        c.migrate_state(entries, vec![]);
        c.finish_migration(0);
    });

    let mut keys = HashMap::new();
    let mut names = HashMap::new();
    let expiry = start_ms + 6 * 24 * 3600 * 1000;
    for (i, name) in sc["users"].as_array().unwrap().iter().enumerate() {
        let name = name.as_str().unwrap().to_string();
        let k = Key::new(10 + i as u8);
        let kh = k.hex();
        c.call_with(acc(&name), NearToken::from_millinear(100), start_ms, |c| c.register_session_key(0, kh, expiry));
        let sub = sub_of(&name, 0);
        keys.insert(sub, k);
        names.insert(sub, name);
    }
    for (i, sys) in [&amm, &ins].iter().enumerate() {
        let k = Key::new(200 + i as u8);
        let (sh, kh) = (sys.to_string(), k.hex());
        c.call_with(owner(), NearToken::from_yoctonear(0), start_ms, |c| c.register_system_session_key(sh, kh, expiry));
        keys.insert(unhex32(sys), k);
    }
    Replay { c, keys, names, nonces: HashMap::new(), now_ms: start_ms, spot_prices: vec![], all_prices: vec![] }
}

fn run_step(r: &mut Replay, st: &Value) -> Result<(), String> {
    let s32 = |k: &str| unhex32(st[k].as_str().unwrap());
    match st["kind"].as_str().unwrap() {
        "tick" => {
            let time = st["time"].as_u64().unwrap();
            r.now_ms = time * 1000;
            let prices = pairs(&st["prices"]);
            r.spot_prices = prices.iter().filter(|(p, _)| p % 2 == 0).copied().collect();
            r.all_prices = prices.clone();
            r.submit(Batch::new().tick(time, pairs(&st["rates"]), prices))
        }
        "match" => {
            let p = pid(&st["pid"]);
            let matched = int(&st["matched"]);
            let (taker, maker) = (s32("taker"), s32("maker"));
            let t = r.order(taker, p, int(&st["takerLimit"]), -matched, st["takerReduce"].as_bool().unwrap_or(false));
            let m = r.order(maker, p, int(&st["price"]), matched, st["makerReduce"].as_bool().unwrap_or(false));
            let b = Batch::new().matched(t, r.key(&taker), m, r.key(&maker), matched);
            r.submit(b)
        }
        "liquidate" => {
            let amount = int(&st["amount"]);
            let lq = s32("liquidator");
            let o = r.order(lq, pid(&st["pid"]), int(&st["price"]), -amount, false);
            let sig = r.key(&lq).sign(&order_digest(&o));
            let payload = tx::Liquidate {
                product_id: pid(&st["pid"]),
                prices: r.all_prices.clone(),
                liquidator: o,
                liquidatee: s32("liquidatee"),
                amount,
            };
            r.submit(Batch::new().push(common::env(tx::LIQUIDATE_SUBACCOUNT, &payload), sig, vec![]))
        }
        "settle" => {
            let subs = st["subs"].as_array().unwrap().iter().map(|v| unhex32(v.as_str().unwrap())).collect();
            let payload = tx::SettleUserPnl { subaccounts: subs, spot_prices: r.spot_prices.clone() };
            r.submit(Batch::new().push(common::env(tx::SETTLE_USER_PNL, &payload), vec![], vec![]))
        }
        "socialise" => {
            let payload = tx::Socialise { subaccount: s32("sub"), spot_prices: r.spot_prices.clone() };
            r.submit(Batch::new().push(common::env(tx::SOCIALISE_SUBACCOUNT, &payload), vec![], vec![]))
        }
        "withdraw" => {
            let sub = s32("sub");
            let nonce = *r.nonces.get(&sub).unwrap_or(&0);
            let w = NearWithdraw {
                subaccount: sub,
                session_key: r.key(&sub).addr,
                product_id: 4,
                amount: int(&st["amount"]) as u128,
                nonce,
                receiver: r.names[&sub].clone(),
            };
            let b = Batch::new().withdraw(w, r.key(&sub));
            let res = r.submit(b);
            if res.is_ok() {
                r.nonces.insert(sub, nonce + 1);
            }
            res
        }
        k => panic!("unknown step {k}"),
    }
}

#[test]
fn go_ledger_parity() {
    // expected rejections panic inside the contract; keep their messages out of the output
    let default_hook = std::panic::take_hook();
    std::panic::set_hook(Box::new(move |info| {
        if !info.to_string().contains("GuestPanic") {
            default_hook(info)
        }
    }));
    let v = load();
    let mut totals: HashMap<String, usize> = HashMap::new();
    for sc in v["scenarios"].as_array().unwrap() {
        let seed = sc["seed"].as_i64().unwrap();
        let mut r = deploy(sc);
        for (i, st) in sc["steps"].as_array().unwrap().iter().enumerate() {
            let want = st["reject"].as_str().unwrap_or("");
            let got = run_step(&mut r, st);
            match (&got, want) {
                (Ok(()), "") => {}
                (Err(msg), w) if !w.is_empty() && msg.contains(w) => {}
                _ => panic!(
                    "seed {seed} step {i} {}: expected {:?}, got {:?}\n{st}",
                    st["kind"],
                    if want.is_empty() { "ok" } else { want },
                    got
                ),
            }
            *totals.entry(format!("{}{}", st["kind"].as_str().unwrap(), if want.is_empty() { "" } else { "/rejected" })).or_default() += 1;
        }

        // final state
        for fin in sc["final"].as_array().unwrap() {
            let sub = unhex32(fin["sub"].as_str().unwrap());
            let got = r.c.view(|c| c.get_subaccount(hex(&sub)));
            let norm = |arr: &Value, width: usize| -> Vec<Vec<i128>> {
                let mut rows: Vec<Vec<i128>> = arr
                    .as_array()
                    .unwrap()
                    .iter()
                    .map(|row| (0..width).map(|j| int(&row[j])).collect::<Vec<_>>())
                    .filter(|row| row[1..].iter().any(|x| *x != 0))
                    .collect();
                rows.sort();
                rows
            };
            assert_eq!(norm(&got["spots"], 2), norm(&fin["spots"], 2), "seed {seed} spots of {}", fin["sub"]);
            assert_eq!(norm(&got["perps"], 4), norm(&fin["perps"], 4), "seed {seed} perps of {}", fin["sub"]);
        }
        let products = r.c.view(|c| c.get_products());
        for oi in sc["openInterest"].as_array().unwrap() {
            let p = products["perps"].as_array().unwrap().iter().find(|x| x["product_id"].as_u64().unwrap() as u32 == pid(&oi[0])).unwrap();
            assert_eq!(int(&p["open_interest_long"]), int(&oi[1]), "seed {seed} OI long {}", oi[0]);
            assert_eq!(int(&p["open_interest_short"]), int(&oi[2]), "seed {seed} OI short {}", oi[0]);
        }
        for cum in sc["cumFunding"].as_array().unwrap() {
            let p =
                products["perps"].as_array().unwrap().iter().find(|x| x["product_id"].as_u64().unwrap() as u32 == pid(&cum[0])).unwrap();
            assert_eq!(int(&p["cum_funding_x18"]), int(&cum[1]), "seed {seed} funding {}", cum[0]);
        }
    }
    println!("parity steps replayed: {totals:?}");
    // the vectors must actually exercise every path
    for k in ["match", "match/rejected", "liquidate", "liquidate/rejected", "settle", "socialise", "withdraw", "withdraw/rejected", "tick"]
    {
        assert!(totals.get(k).copied().unwrap_or(0) > 0, "no {k} steps in vectors");
    }
}
