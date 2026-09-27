//! Go (nearchain/payloads.go) and the contract agree on every batch payload and every signed request.
//! vectors/payloads.json: decode each Go encoding with the contract's own types, compare every
//! field, re-encode, and require identical bytes. vectors/requests.json: recompute each EIP-712
//! digest with the contract's hashing and recover the session key from Go's signature.
use near_sdk::borsh::{self, BorshDeserialize, BorshSerialize};
use near_stocks_core::eip712::{self, *};
use near_stocks_core::tx::{self, *};
use serde_json::Value;

fn load(name: &str) -> Value {
    let p = format!("{}/../../vectors/{name}", env!("CARGO_MANIFEST_DIR"));
    serde_json::from_str(&std::fs::read_to_string(&p).expect(&p)).unwrap()
}
fn unhex(s: &str) -> Vec<u8> {
    hex::decode(s.trim_start_matches("0x")).unwrap()
}
fn arr<const N: usize>(v: &Value) -> [u8; N] {
    unhex(v.as_str().unwrap()).try_into().unwrap()
}
fn i(v: &Value) -> i128 {
    v.as_str().map(|s| s.parse().unwrap()).unwrap_or_else(|| v.as_i64().unwrap() as i128)
}
fn u(v: &Value) -> u128 {
    v.as_str().map(|s| s.parse().unwrap()).unwrap_or_else(|| v.as_u64().unwrap() as u128)
}
fn pairs(v: &Value) -> Vec<(u32, i128)> {
    v.as_array().unwrap().iter().map(|p| (i(&p[0]) as u32, i(&p[1]))).collect()
}
fn order(v: &Value) -> Order {
    Order {
        subaccount: arr(&v["subaccount"]),
        price_x18: i(&v["priceX18"]),
        amount: i(&v["amount"]),
        expiration: v["expiration"].as_u64().unwrap(),
        is_reduce: v["isReduce"].as_bool().unwrap(),
        session_key: arr(&v["sessionKey"]),
        product_id: v["productId"].as_u64().unwrap() as u32,
    }
}

/// decode(bytes) must equal `want`, and encode(want) must equal bytes.
fn roundtrip<T: BorshSerialize + BorshDeserialize + PartialEq + std::fmt::Debug>(name: &str, bytes: &[u8], ty: u8, want: T) {
    assert_eq!(bytes[0], ty, "{name}: type byte");
    let got: T = borsh::from_slice(&bytes[1..]).unwrap_or_else(|e| panic!("{name}: decode {e}"));
    assert_eq!(got, want, "{name}: fields");
    let mut re = vec![ty];
    re.extend(borsh::to_vec(&want).unwrap());
    assert_eq!(re, bytes, "{name}: re-encoding");
}

#[test]
fn every_go_payload_decodes_and_reencodes_identically() {
    let v = load("payloads.json");
    let mut seen = 0;
    for c in v["cases"].as_array().unwrap() {
        let name = c["name"].as_str().unwrap();
        let b = unhex(c["hex"].as_str().unwrap());
        let f = &c["fields"];
        let ty = c["type"].as_u64().unwrap() as u8;
        let withdraw = || NearWithdraw {
            subaccount: arr(&f["subaccount"]),
            session_key: arr(&f["sessionKey"]),
            product_id: f["productId"].as_u64().unwrap() as u32,
            amount: u(&f["amount"]),
            nonce: u(&f["nonce"]),
            receiver: f["receiver"].as_str().unwrap().into(),
        };
        let pool = || PoolTrade {
            order: PoolOrder {
                subaccount: arr(&f["subaccount"]),
                product_id: f["productId"].as_u64().unwrap() as u32,
                amount: i(&f["amount"]),
                is_buy: f["isBuy"].as_bool().unwrap(),
                nonce: u(&f["nonce"]),
                session_key: arr(&f["sessionKey"]),
            },
            quote_delta: i(&f["quoteDelta"]),
            fees: i(&f["fees"]),
        };
        match name {
            "perp_tick" | "perp_tick_empty" => roundtrip(
                name,
                &b,
                tx::PERPTICK,
                PerpTick { time: f["time"].as_u64().unwrap(), rates: pairs(&f["rates"]), prices: pairs(&f["prices"]) },
            ),
            "match_orders" => roundtrip(
                name,
                &b,
                tx::MATCH_ORDERS,
                MatchOrders {
                    product_id: f["productId"].as_u64().unwrap() as u32,
                    taker: order(&f["taker"]),
                    maker: order(&f["maker"]),
                    matched_amount: i(&f["matchedAmount"]),
                },
            ),
            "liquidate" => roundtrip(
                name,
                &b,
                tx::LIQUIDATE_SUBACCOUNT,
                Liquidate {
                    product_id: f["productId"].as_u64().unwrap() as u32,
                    prices: pairs(&f["prices"]),
                    liquidator: order(&f["liquidator"]),
                    liquidatee: arr(&f["liquidatee"]),
                    amount: i(&f["amount"]),
                },
            ),
            "withdraw_collateral" => roundtrip(name, &b, tx::WITHDRAW_COLLATERAL, withdraw()),
            // This contract carries no LogX token, staking or rewards pool: it doesn't implement
            // these transaction types at all (Go's payloads.json still generates a reference
            // encoding for them, since the byte format itself is unaffected). Not roundtripped or
            // counted in `seen` below.
            "withdraw_logx" | "stake_logx" | "unstake_logx" | "claim_rewards" | "claim_logx" | "reward_rate_tick" => continue,
            "settle_user_pnl" => roundtrip(
                name,
                &b,
                tx::SETTLE_USER_PNL,
                SettleUserPnl {
                    subaccounts: f["subaccounts"].as_array().unwrap().iter().map(arr).collect(),
                    spot_prices: pairs(&f["spotPrices"]),
                },
            ),
            "socialise" => roundtrip(
                name,
                &b,
                tx::SOCIALISE_SUBACCOUNT,
                Socialise { subaccount: arr(&f["subaccount"]), spot_prices: pairs(&f["spotPrices"]) },
            ),
            "set_nonce" => {
                roundtrip(name, &b, tx::SET_NONCE, SetNonce { subaccount: arr(&f["subaccount"]), delta: f["delta"].as_u64().unwrap() })
            }
            "place_option" => roundtrip(
                name,
                &b,
                tx::PLACE_OPTIONS_BET,
                PlaceOption {
                    bet: OptionBet {
                        subaccount: arr(&f["subaccount"]),
                        product_id: f["productId"].as_u64().unwrap() as u32,
                        amount: i(&f["amount"]),
                        interval: f["interval"].as_u64().unwrap() as u32,
                        nonce: u(&f["nonce"]),
                        session_key: arr(&f["sessionKey"]),
                    },
                    order_id: f["orderId"].as_u64().unwrap(),
                    entry_price_x18: i(&f["entryPriceX18"]),
                    payout_pct: f["payoutPct"].as_u64().unwrap() as u32,
                    fee_pct: f["feePct"].as_u64().unwrap() as u32,
                },
            ),
            "close_option" => roundtrip(
                name,
                &b,
                tx::CLOSE_OPTIONS_BET,
                CloseOption { order_id: f["orderId"].as_u64().unwrap(), exit_price_x18: i(&f["exitPriceX18"]) },
            ),
            "pre_market_order" => roundtrip(name, &b, tx::PRE_MARKET_ORDER_REQUEST, pool()),
            "synthetic_spot_order" => roundtrip(name, &b, tx::SYN_SPOT_ORDER_REQUEST, pool()),
            other => panic!("unknown case {other}"),
        }
        assert_eq!(ty, b[0]);
        seen += 1;
    }
    // 18 cases in payloads.json, minus the 6 LogX/rewards ones this contract doesn't implement.
    assert_eq!(seen, 12);

    // submit_transactions arguments: borsh((u64, Vec<Vec<u8>>, Vec<Vec<u8>>, Vec<Vec<u8>>))
    let s = &v["submit"];
    let list = |k: &str| s[k].as_array().unwrap().iter().map(|x| unhex(x.as_str().unwrap())).collect::<Vec<_>>();
    let want = (s["idx"].as_u64().unwrap(), list("txs"), list("sigs"), list("sigs2"));
    assert_eq!(borsh::to_vec(&want).unwrap(), unhex(s["hex"].as_str().unwrap()));
}

#[test]
fn every_go_request_signature_verifies_in_the_contract() {
    let v = load("requests.json");
    let chain = v["chainId"].as_u64().unwrap();
    let sep = eip712::domain_separator(chain, &eip712::verifying_contract(v["contractAccount"].as_str().unwrap()));
    let key: [u8; 20] = arr(&v["sessionKey"]);
    let sub: [u8; 32] = arr(&v["subAccountId"]);
    let mut seen = 0;
    for r in v["requests"].as_array().unwrap() {
        let p = r["primaryType"].as_str().unwrap();
        // This contract carries no LogX token, staking or rewards pool: it doesn't implement these
        // request types at all, so there's no hash to recompute or verify here.
        if matches!(p, "StakeLogXRequest" | "UnstakeLogXRequest" | "ClaimRewards" | "ClaimLogX") {
            continue;
        }
        let hash = match p {
            "UserOptionBet" => option_bet_hash(
                &OptionBet {
                    subaccount: sub,
                    product_id: i(&r["productId"]) as u32,
                    amount: i(&r["amount"]),
                    interval: i(&r["interval"]) as u32,
                    nonce: u(&r["nonce"]),
                    session_key: key,
                },
                chain,
            ),
            "PlacePreMarketOrderRequest" | "PlaceSyntheticSpotOrderRequest" => {
                let ty = if p == "PlacePreMarketOrderRequest" { PRE_MARKET_ORDER_TYPE } else { SYN_SPOT_ORDER_TYPE };
                pool_order_hash(
                    ty,
                    &PoolOrder {
                        subaccount: sub,
                        product_id: i(&r["productId"]) as u32,
                        amount: i(&r["amount"]),
                        is_buy: r["isBuy"].as_bool().unwrap(),
                        nonce: u(&r["nonce"]),
                        session_key: key,
                    },
                    chain,
                )
            }
            other => panic!("unknown type {other}"),
        };
        let digest = typed_digest(&sep, &hash);
        assert_eq!(format!("0x{}", hex::encode(digest)), r["digest"].as_str().unwrap(), "{p}: digest");
        let signer = recover_signer(&digest, &unhex(r["signature"].as_str().unwrap())).unwrap();
        assert_eq!(signer, key, "{p}: signer");
        seen += 1;
    }
    // 7 requests in requests.json, minus the 4 LogX/rewards ones this contract doesn't implement.
    assert_eq!(seen, 3);
}
