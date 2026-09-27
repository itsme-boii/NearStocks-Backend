//! Integration tests on a real NEAR sandbox node (Development.md §13.1) and gate G2 measurements.
//! Needs both wasm files: `cargo near build non-reproducible-wasm` in core/ and in mock-ft/.
//! Run through scripts/sandbox-test.sh (the sandbox binary needs glibc >= 2.38).
mod common;
use common::{Key, BTC, CHAIN_ID, E18, USDC};
use near_sdk::borsh;
use near_stocks_core::eip712::{NearWithdraw, Order};
use near_stocks_core::events::hex;
use near_stocks_core::{identity, tx};
use near_workspaces::result::ExecutionFinalResult;
use near_workspaces::types::{Gas, NearToken};
use near_workspaces::{Account, Contract, Worker};
use serde_json::{json, Value};
use std::time::{SystemTime, UNIX_EPOCH};

const CORE_WASM: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/../target/near/near_stocks_core/near_stocks_core.wasm");
const FT_WASM: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/../target/near/mock_ft/mock_ft.wasm");
const P65K: i128 = 65_000 * E18;

fn now_ms() -> u64 {
    SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_millis() as u64
}

fn tgas(g: Gas) -> f64 {
    g.as_gas() as f64 / 1e12
}

struct Env {
    worker: Worker<near_workspaces::network::Sandbox>,
    core: Contract,
    usdc: Contract,
    owner: Account,
    guardian: Account,
    sequencer: Account,
    seq_idx: u64,
    order_seq: u64,
}

impl Env {
    async fn new() -> anyhow::Result<Env> {
        let worker = near_workspaces::sandbox().await?;
        let root = worker.root_account()?;
        let mk = |name: &'static str, near: u128| {
            let root = root.clone();
            async move {
                root.create_subaccount(name)
                    .initial_balance(NearToken::from_near(near))
                    .transact()
                    .await?
                    .into_result()
                    .map_err(anyhow::Error::from)
            }
        };
        let core_acc = mk("core", 30).await?;
        let usdc_acc = mk("usdc", 10).await?;
        let (owner, guardian, sequencer) = (mk("dao", 20).await?, mk("guardian", 5).await?, mk("sequencer", 50).await?);
        let core = core_acc.deploy(&std::fs::read(CORE_WASM)?).await?.into_result()?;
        let usdc = usdc_acc.deploy(&std::fs::read(FT_WASM)?).await?.into_result()?;
        usdc.call("new").transact().await?.into_result()?;
        let sub = |b: [u8; 32]| hex(&b);
        core.call("new")
            .args_json(json!({
                "owner": owner.id(), "guardian": guardian.id(), "sequencer": sequencer.id(), "chain_id": CHAIN_ID, "broker_id": 2,
                "amm_subaccount": sub(common::amm_sub()), "insurance_subaccount": sub(common::insurance_sub()), "fee_subaccount": sub(common::fee_sub()),
            }))
            .transact()
            .await?
            .into_result()?;
        let e = Env { worker, core, usdc, owner, guardian, sequencer, seq_idx: 0, order_seq: 0 };
        e.owner_call(
            "upsert_spot",
            json!({ "product_id": USDC, "config": {
            "token": e.usdc.id(), "decimals": 6, "weighted": true, "withdraw_fee_x18": (E18 / 2).to_string(),
            "price_x18": E18.to_string(), "max_deviation_bps": 200 } }),
        )
        .await?;
        e.owner_call(
            "upsert_perp",
            json!({ "product_id": BTC, "config": {
            "imf_x18": (E18 / 10).to_string(), "mmf_x18": (E18 / 20).to_string(), "liq_frac_x18": (15 * E18 / 1000).to_string(),
            "price_x18": P65K.to_string(), "max_deviation_bps": 1000, "amm_max_position_x18": "0" } }),
        )
        .await?;
        e.owner_call("set_product_order", json!({ "spot_order": [USDC], "collateral_order": [USDC] })).await?;
        e.owner_call("finish_migration", json!({ "n_submissions": 0 })).await?;
        // the contract itself must hold USDC storage to receive transfers
        e.usdc.call("storage_deposit").args_json(json!({ "account_id": e.core.id() })).transact().await?.into_result()?;
        Ok(e)
    }

    async fn owner_call(&self, m: &str, args: Value) -> anyhow::Result<()> {
        let r = self.owner.call(self.core.id(), m).args_json(args).max_gas().transact().await?;
        anyhow::ensure!(r.is_success(), "{m} failed: {:?}", r.failures());
        Ok(())
    }

    async fn user(&self, name: &str, usdc_units: u128, key: &Key) -> anyhow::Result<Account> {
        let root = self.worker.root_account()?;
        let a = root.create_subaccount(name).initial_balance(NearToken::from_near(5)).transact().await?.into_result()?;
        self.usdc.call("storage_deposit").args_json(json!({ "account_id": a.id() })).transact().await?.into_result()?;
        self.usdc
            .call("mint")
            .args_json(json!({ "account_id": a.id(), "amount": usdc_units.to_string() }))
            .transact()
            .await?
            .into_result()?;
        let r = a
            .call(self.core.id(), "register_session_key")
            .args_json(json!({ "subaccount_number": 1, "session_key": key.hex(), "expiry_ms": now_ms() + 24 * 3600 * 1000 }))
            .deposit(NearToken::from_millinear(50))
            .transact()
            .await?;
        anyhow::ensure!(r.is_success(), "register failed: {:?}", r.failures());
        Ok(a)
    }

    async fn deposit(&self, from: &Account, units: u128, msg: &str) -> anyhow::Result<ExecutionFinalResult> {
        let r = from
            .call(self.usdc.id(), "ft_transfer_call")
            .args_json(json!({ "receiver_id": self.core.id(), "amount": units.to_string(), "msg": msg }))
            .deposit(NearToken::from_yoctonear(1))
            .max_gas()
            .transact()
            .await?;
        anyhow::ensure!(r.is_success(), "deposit failed: {:?}", r.failures());
        Ok(r)
    }

    fn sub(&self, a: &Account) -> [u8; 32] {
        identity::subaccount_id(2, a.id().as_str(), 1) // the app's subaccount (`2_<addr>_1`)
    }

    fn order(&mut self, sub: [u8; 32], k: &Key, price: i128, amount: i128) -> Order {
        self.order_seq += 1;
        Order {
            subaccount: sub,
            price_x18: price,
            amount,
            expiration: now_ms() + 3_600_000 + self.order_seq,
            is_reduce: false,
            session_key: k.addr,
            product_id: BTC,
        }
    }

    fn match_tx(&mut self, taker: [u8; 32], tk: &Key, maker: [u8; 32], mk: &Key, amt: i128) -> (Vec<u8>, Vec<u8>, Vec<u8>) {
        let t = self.order(taker, tk, 0, amt);
        let m = self.order(maker, mk, P65K, -amt);
        let id = self.core.id().to_string();
        let (ts, ms) = (tk.sign(&common::order_digest_in(&id, &t)), mk.sign(&common::order_digest_in(&id, &m)));
        (common::env(tx::MATCH_ORDERS, &tx::MatchOrders { product_id: BTC, taker: t, maker: m, matched_amount: -amt }), ts, ms)
    }

    fn tick_tx(&self, prices: Vec<(u32, i128)>) -> (Vec<u8>, Vec<u8>, Vec<u8>) {
        let t = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() - 2;
        (common::env(tx::PERPTICK, &tx::PerpTick { time: t, rates: vec![], prices }), vec![], vec![])
    }

    fn withdraw_tx(&self, sub: [u8; 32], k: &Key, amount: u128, nonce: u128, receiver: &str) -> (Vec<u8>, Vec<u8>, Vec<u8>) {
        let w = NearWithdraw { subaccount: sub, session_key: k.addr, product_id: USDC, amount, nonce, receiver: receiver.into() };
        let sig = k.sign(&common::withdraw_digest_in(self.core.id().as_str(), &w));
        (common::env(tx::WITHDRAW_COLLATERAL, &w), sig, vec![])
    }

    /// Sends one batch; on success the local index advances.
    async fn submit(&mut self, txs: Vec<(Vec<u8>, Vec<u8>, Vec<u8>)>) -> anyhow::Result<ExecutionFinalResult> {
        let n = txs.len() as u64;
        let (a, b, c): (Vec<_>, Vec<_>, Vec<_>) = txs.into_iter().fold((vec![], vec![], vec![]), |mut acc, (x, y, z)| {
            acc.0.push(x);
            acc.1.push(y);
            acc.2.push(z);
            acc
        });
        let args = borsh::to_vec(&(self.seq_idx, a, b, c))?;
        let r = self.sequencer.call(self.core.id(), "submit_transactions").args(args).max_gas().transact().await?;
        if r.is_success() {
            self.seq_idx += n;
        }
        Ok(r)
    }

    async fn spot(&self, sub: [u8; 32]) -> anyhow::Result<i128> {
        let v: Value = self.core.view("get_subaccount").args_json(json!({ "subaccount": hex(&sub) })).await?.json()?;
        Ok(v["spots"].as_array().unwrap().iter().find(|e| e[0] == USDC).map(|e| e[1].as_str().unwrap().parse().unwrap()).unwrap_or(0))
    }

    async fn ft_balance(&self, a: &str) -> anyhow::Result<u128> {
        let v: String = self.usdc.view("ft_balance_of").args_json(json!({ "account_id": a })).await?.json()?;
        Ok(v.parse()?)
    }

    async fn n_submissions(&self) -> anyhow::Result<u64> {
        Ok(self.core.view("n_submissions").await?.json()?)
    }
}

fn logs_contain(r: &ExecutionFinalResult, needle: &str) -> bool {
    r.logs().iter().any(|l| l.contains(needle))
}

fn failure_text(r: &ExecutionFinalResult) -> String {
    format!("{:?}", r.failures())
}

#[tokio::test]
#[ignore = "needs sandbox binary + built wasm"]
async fn sandbox_end_to_end() -> anyhow::Result<()> {
    let mut e = Env::new().await?;
    let (ka, kb) = (Key::new(1), Key::new(2));
    let alice = e.user("alice", 10_000_000_000, &ka).await?; // 10,000 USDC
    let bob = e.user("bob", 10_000_000_000, &kb).await?;
    let (sa, sb) = (e.sub(&alice), e.sub(&bob));

    // --- deposits through the real NEP-141 flow (§7.1)
    let r = e.deposit(&alice, 5_000_000_000, "").await?;
    assert!(logs_contain(&r, "\"event\":\"deposit\""));
    let _ = e.deposit(&bob, 5_000_000_000, "").await?;
    assert_eq!(e.spot(sa).await?, 5_000 * E18);
    assert_eq!(e.ft_balance(e.core.id().as_str()).await?, 10_000_000_000);

    // deposit while paused is refunded by ft_resolve_transfer
    e.guardian.call(e.core.id(), "pause").args_json(json!({ "mask": 2 })).transact().await?.into_result()?;
    let _ = e.deposit(&alice, 1_000_000, "").await?;
    assert_eq!(e.ft_balance(alice.id().as_str()).await?, 5_000_000_000, "paused deposit refunded");
    e.owner_call("unpause", json!({ "mask": 2 })).await?;

    // a token the contract does not accept is refunded in full (ft_on_transfer panics)
    let fake =
        e.worker.root_account()?.create_subaccount("fake").initial_balance(NearToken::from_near(10)).transact().await?.into_result()?;
    let fake = fake.deploy(&std::fs::read(FT_WASM)?).await?.into_result()?;
    fake.call("new").transact().await?.into_result()?;
    for a in [alice.id(), e.core.id()] {
        fake.call("storage_deposit").args_json(json!({ "account_id": a })).transact().await?.into_result()?;
    }
    fake.call("mint").args_json(json!({ "account_id": alice.id(), "amount": "500" })).transact().await?.into_result()?;
    let _ = alice
        .call(fake.id(), "ft_transfer_call")
        .args_json(json!({ "receiver_id": e.core.id(), "amount": "500", "msg": "" }))
        .deposit(NearToken::from_yoctonear(1))
        .max_gas()
        .transact()
        .await?;
    let back: String = fake.view("ft_balance_of").args_json(json!({ "account_id": alice.id() })).await?.json()?;
    assert_eq!(back, "500");

    // --- a batch: price tick + match (§5.4)
    let m1 = e.match_tx(sa, &ka, sb, &kb, E18 / 10);
    let r = e.submit(vec![e.tick_tx(vec![(BTC, P65K), (USDC, E18)]), m1]).await?;
    assert!(r.is_success(), "{}", failure_text(&r));
    assert!(logs_contain(&r, "\"fills\""));
    assert_eq!(e.spot(sa).await?, 5_000 * E18 - 39 * E18 / 10);
    assert_eq!(e.n_submissions().await?, 2);

    // --- atomicity: a bad transaction reverts the good one before it
    let good = e.match_tx(sa, &ka, sb, &kb, E18 / 10);
    let mut bad = e.match_tx(sa, &ka, sb, &kb, E18 / 10);
    bad.1 = kb.sign(&[7u8; 32]);
    let r = e.submit(vec![good, bad]).await?;
    assert!(!r.is_success());
    assert!(failure_text(&r).contains("signature does not match session key"), "{}", failure_text(&r));
    assert_eq!(e.n_submissions().await?, 2);
    assert_eq!(e.spot(sa).await?, 5_000 * E18 - 39 * E18 / 10, "reverted batch changed nothing");

    // only the sequencer may submit
    let r = alice
        .call(e.core.id(), "submit_transactions")
        .args(borsh::to_vec(&(2u64, vec![vec![0u8]], vec![Vec::<u8>::new()], vec![Vec::<u8>::new()]))?)
        .transact()
        .await?;
    assert!(failure_text(&r).contains("only sequencer"));

    // --- withdrawal: debit, ft_transfer, callback moves the fee (§5.8)
    let before_ft = e.ft_balance(alice.id().as_str()).await?;
    let r = e.submit(vec![e.withdraw_tx(sa, &ka, 100 * E18 as u128, 0, alice.id().as_str())]).await?;
    assert!(r.is_success(), "{}", failure_text(&r));
    assert!(logs_contain(&r, "withdraw_pending") && logs_contain(&r, "withdraw_done"), "{:?}", r.logs());
    assert_eq!(e.ft_balance(alice.id().as_str()).await? - before_ft, 99_500_000, "100 USDC minus 0.5 fee");
    assert_eq!(e.spot(common::fee_sub()).await?, 78 * E18 / 10 + E18 / 2);

    // forced ft_transfer failure: the callback re-credits the whole amount
    e.usdc.call("set_fail").args_json(json!({ "account_id": alice.id(), "fail": true })).transact().await?.into_result()?;
    let bal = e.spot(sa).await?;
    let r = e.submit(vec![e.withdraw_tx(sa, &ka, 50 * E18 as u128, 1, alice.id().as_str())]).await?;
    assert!(r.is_success(), "the batch itself succeeds: {}", failure_text(&r));
    assert!(logs_contain(&r, "withdraw_failed"), "{:?}", r.logs());
    assert_eq!(e.spot(sa).await?, bal, "re-credited");
    e.usdc.call("set_fail").args_json(json!({ "account_id": alice.id(), "fail": false })).transact().await?.into_result()?;

    // receiver without USDC storage (S-7): same re-credit path
    let r = e.submit(vec![e.withdraw_tx(sa, &ka, 20 * E18 as u128, 2, "nobody.test.near")]).await?;
    assert!(logs_contain(&r, "withdraw_failed"));
    assert_eq!(e.spot(sa).await?, bal);

    // the callback is private
    let r = alice
        .call(e.core.id(), "on_withdraw_complete")
        .args_json(json!({ "subaccount": hex(&sa), "product_id": USDC, "amount_x18": "1", "fee_x18": "0", "payout": "1", "receiver": alice.id(), "tx_idx": 0 }))
        .transact()
        .await?;
    assert!(failure_text(&r).contains("private"), "{}", failure_text(&r));

    // pause withdrawals: the batch is rejected
    e.guardian.call(e.core.id(), "pause").args_json(json!({ "mask": 4 })).transact().await?.into_result()?;
    let r = e.submit(vec![e.withdraw_tx(sa, &ka, 10 * E18 as u128, 3, alice.id().as_str())]).await?;
    assert!(failure_text(&r).contains("paused"));
    let r = e.guardian.call(e.core.id(), "unpause").args_json(json!({ "mask": 4 })).transact().await?;
    assert!(failure_text(&r).contains("only owner"));
    e.owner_call("unpause", json!({ "mask": 4 })).await?;

    // --- D-7: the DAO sweeps fees to a treasury account (1 yocto, owner only)
    let treasury =
        e.worker.root_account()?.create_subaccount("treasury").initial_balance(NearToken::from_near(2)).transact().await?.into_result()?;
    e.usdc.call("storage_deposit").args_json(json!({ "account_id": treasury.id() })).transact().await?.into_result()?;
    let fees = e.spot(common::fee_sub()).await?;
    let r = alice
        .call(e.core.id(), "sweep_fees")
        .args_json(json!({ "product_id": USDC, "amount_x18": (4 * E18).to_string(), "receiver": treasury.id() }))
        .deposit(NearToken::from_yoctonear(1))
        .max_gas()
        .transact()
        .await?;
    assert!(failure_text(&r).contains("only owner"));
    let r = e
        .owner
        .call(e.core.id(), "sweep_fees")
        .args_json(json!({ "product_id": USDC, "amount_x18": (4 * E18).to_string(), "receiver": treasury.id() }))
        .deposit(NearToken::from_yoctonear(1))
        .max_gas()
        .transact()
        .await?;
    assert!(r.is_success(), "{}", failure_text(&r));
    assert!(logs_contain(&r, "fee_sweep_pending") && logs_contain(&r, "withdraw_done"), "{:?}", r.logs());
    assert_eq!(e.ft_balance(treasury.id().as_str()).await?, 4_000_000);
    assert_eq!(e.spot(common::fee_sub()).await?, fees - 4 * E18);

    // --- upgrade keeps state (§5.10)
    let before = e.n_submissions().await?;
    let r = e.owner.call(e.core.id(), "upgrade").args(std::fs::read(CORE_WASM)?).max_gas().transact().await?;
    assert!(r.is_success(), "{}", failure_text(&r));
    assert_eq!(e.n_submissions().await?, before);
    assert_eq!(e.spot(sa).await?, bal);
    let r = alice.call(e.core.id(), "upgrade").args(vec![0u8; 8]).max_gas().transact().await?;
    assert!(failure_text(&r).contains("only owner"));
    Ok(())
}

/// Gate G2: gas per transaction type and batch sizes, storage per subaccount.
#[tokio::test]
#[ignore = "needs sandbox binary + built wasm"]
async fn sandbox_gas_and_storage() -> anyhow::Result<()> {
    let mut e = Env::new().await?;
    let n_users = 8;
    let mut users = vec![];
    let storage_before = e.worker.view_account(e.core.id()).await?.storage_usage;
    for i in 0..n_users {
        let k = Key::new(50 + i as u8);
        let a = e.user(&format!("u{i}"), 1_000_000_000_000, &k).await?;
        let _ = e.deposit(&a, 500_000_000_000, "").await?;
        users.push((e.sub(&a), k, a));
    }
    let storage_after = e.worker.view_account(e.core.id()).await?.storage_usage;
    let per_user = (storage_after - storage_before) as f64 / n_users as f64;
    println!("G2 storage: {per_user:.0} bytes per subaccount (session key + owner + USDC balance)");

    // the contract's own receipt (the first receipt after the transaction converts to it)
    let measure = |r: &ExecutionFinalResult| tgas(r.receipt_outcomes()[0].gas_burnt);
    let r = e.submit(vec![e.tick_tx(vec![(BTC, P65K), (USDC, E18)])]).await?;
    assert!(r.is_success(), "{}", failure_text(&r));
    println!("G2 PERPTICK (2 prices): {:.2} Tgas", measure(&r));

    let mut results = vec![];
    for n in [1usize, 10, 25, 50, 80] {
        let txs: Vec<_> = (0..n)
            .map(|i| {
                let (t, m) = (i % n_users, (i + 1) % n_users);
                let (ts, tk) = (users[t].0, Key::new(50 + t as u8));
                let (ms, mk) = (users[m].0, Key::new(50 + m as u8));
                e.match_tx(ts, &tk, ms, &mk, E18 / 1000)
            })
            .collect();
        let st0 = e.worker.view_account(e.core.id()).await?.storage_usage;
        let r = e.submit(txs).await?;
        assert!(r.is_success(), "batch of {n}: {}", failure_text(&r));
        if n == 80 {
            let st1 = e.worker.view_account(e.core.id()).await?.storage_usage;
            println!("G2 storage: {:.0} bytes per filled order (replay-protection entry)", (st1 - st0) as f64 / (2 * n) as f64);
        }
        let g = measure(&r);
        let log_bytes: usize = r.logs().iter().map(|l| l.len()).sum();
        println!("G2 MATCH_ORDERS batch of {n}: {g:.1} Tgas ({:.2} per match), {log_bytes} log bytes", g / n as f64);
        results.push((n, g));
    }
    let (n1, g1) = results[0];
    let (n2, g2) = *results.last().unwrap();
    let marginal = (g2 - g1) / (n2 - n1) as f64;
    let base = g1 - marginal * n1 as f64;
    let max_at_240 = ((240.0 - base) / marginal).floor();
    println!("G2 MATCH marginal {marginal:.2} Tgas, base {base:.2} Tgas -> max {max_at_240} matches per batch at 240 Tgas (20% headroom)");

    let (s, k, a) = &users[0];
    let r = e.submit(vec![e.withdraw_tx(*s, k, 10 * E18 as u128, 0, a.id().as_str())]).await?;
    assert!(r.is_success(), "{}", failure_text(&r));
    println!(
        "G2 WITHDRAW_COLLATERAL: {:.2} Tgas in the batch receipt (+{} Tgas reserved for ft_transfer and callback), {:.2} Tgas total",
        measure(&r),
        25,
        tgas(r.total_gas_burnt)
    );
    let r = e.deposit(a, 1_000_000, "").await?;
    println!("G2 deposit (ft_transfer_call end to end): {:.2} Tgas total", tgas(r.total_gas_burnt));
    let settle = tx::SettleUserPnl { subaccounts: users.iter().map(|u| u.0).collect(), spot_prices: vec![(USDC, E18)] };
    let r = e.submit(vec![(common::env(tx::SETTLE_USER_PNL, &settle), vec![], vec![])]).await?;
    assert!(r.is_success(), "{}", failure_text(&r));
    println!("G2 SETTLE_USER_PNL ({n_users} subaccounts, none negative): {:.2} Tgas", measure(&r));
    assert!(max_at_240 >= 50.0, "old BATCH_SIZE of 50 must fit with headroom");
    Ok(())
}
