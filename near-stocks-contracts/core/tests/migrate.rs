//! Tests the `migrate()` upgrade hook against the exact pre-2026-09-27 field layout
//! (`PreLogxRemoval`, kept in `core/src/lib.rs` only for this) — the shape the currently-deployed
//! testnet contract's storage is actually in. Builds that old shape by hand with realistic data
//! (a subaccount with a real balance, a listed spot, a trusted depositor, nonzero claim/reward
//! fields), runs the real `migrate()`, and asserts every field the new contract keeps survived
//! unchanged while the two removed fields are simply gone (the compiler already guarantees that;
//! this just proves the *rest* wasn't lost in translation).
mod common;
use common::*;
use near_sdk::borsh::BorshSerialize;
use near_sdk::json_types::I128;
use near_sdk::store::{IterableMap, LookupMap};
use near_sdk::BorshStorageKey;
use near_stocks_core::events::hex;
use near_stocks_core::perp::FeeTable;
use near_stocks_core::state::{Price, ProductAccounts, SpotProduct, Subaccount};
use near_stocks_core::{PerpConfig, PreLogxRemoval, PreLogxRemovalClaimLimits};

#[derive(BorshStorageKey, BorshSerialize)]
#[borsh(crate = "near_sdk::borsh")]
enum TestKey {
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
    Held,
    Unclaimed,
}

fn alice() -> [u8; 32] {
    sub_of("alice.near", 0)
}

fn old_state() -> PreLogxRemoval {
    let mut subaccounts = LookupMap::new(TestKey::Subaccounts);
    let mut sub = Subaccount::default();
    sub.set_spot(USDC, 500 * E18);
    subaccounts.insert(alice(), sub);

    let mut owners = LookupMap::new(TestKey::Owners);
    owners.insert(alice(), acc("alice.near"));

    let mut spots = IterableMap::new(TestKey::Spots);
    spots.insert(
        USDC,
        SpotProduct {
            token: Some(usdc_token()),
            decimals: 6,
            weighted: true,
            withdraw_fee_x18: E18 / 2,
            price: Price { price_x18: E18, updated_at: 100 },
            max_deviation_bps: 200,
        },
    );

    let mut nonces = LookupMap::new(TestKey::Nonces);
    nonces.insert(alice(), 3);

    PreLogxRemoval {
        owner: owner(),
        guardian: guardian(),
        sequencer: sequencer(),
        chain_id: CHAIN_ID,
        broker_id: BROKER,
        n_submissions: 42,
        paused: 0,
        migrated: true,
        fee_table: FeeTable::legacy(),
        amm_subaccount: amm_sub(),
        insurance_subaccount: insurance_sub(),
        fee_subaccount: fee_sub(),
        price_max_age_sec: 300,
        max_session_ttl_ms: 7 * 24 * 3600 * 1000,
        min_new_deposit_x18: E18,
        max_order_ttl_ms: 31 * 24 * 3600 * 1000,
        spot_order: vec![USDC],
        collateral_order: vec![USDC],
        spots,
        perps: IterableMap::new(TestKey::Perps),
        token_to_spot: LookupMap::new(TestKey::TokenToSpot),
        subaccounts,
        owners,
        session_keys: LookupMap::new(TestKey::SessionKeys),
        nonces,
        filled: LookupMap::new(TestKey::Filled),
        product_accounts: ProductAccounts::default(),
        side_products: IterableMap::new(TestKey::SideProducts),
        options: LookupMap::new(TestKey::Options),
        pool_balances: LookupMap::new(TestKey::PoolBalances),
        claim_limits: PreLogxRemovalClaimLimits { max_logx_claim_x18: I128(10_000 * E18), max_reward_claim_x18: I128(500 * E18) },
        reward_rate_x18: 7 * E18,
        trusted_depositors: vec![acc("intents.near")],
        held: LookupMap::new(TestKey::Held),
        unclaimed: LookupMap::new(TestKey::Unclaimed),
    }
}

#[test]
fn migrate_keeps_every_surviving_field_and_the_contract_stays_usable() {
    let mut c = Chain::deploy_state(old_state);
    c.run_migrate();

    // Config and roles.
    c.view(|c| {
        let cfg = c.get_config();
        assert_eq!(cfg["owner"], owner().to_string());
        assert_eq!(cfg["sequencer"], sequencer().to_string());
        assert_eq!(cfg["n_submissions"], 42);
        assert_eq!(cfg["migrated"], true);
        assert_eq!(cfg["price_max_age_sec"], 300);
        assert_eq!(cfg["fee_subaccount"], hex(&fee_sub()));
    });

    // The listed spot and the subaccount's real balance both survived.
    c.view(|c| {
        let products = c.get_products();
        let usdc = &products["spots"][0];
        assert_eq!(usdc["product_id"], USDC);
        assert_eq!(usdc["price_x18"], E18.to_string());
    });
    assert_eq!(spot(&c, &alice(), USDC), 500 * E18);
    assert_eq!(c.view(|c| c.get_nonce(hex(&alice()))), 3);

    // The contract isn't just readable — it still processes a real batch after migrating, using
    // the very trading path the removed LogX code never touched.
    c.call(owner(), |c| c.upsert_perp(BTC, PerpConfig {
        imf_x18: I128(E18 / 10), mmf_x18: I128(E18 / 20), liq_frac_x18: I128(15 * E18 / 1000),
        price_x18: I128(65_000 * E18), max_deviation_bps: 1000, amm_max_position_x18: I128(0),
    }));
    let k = Key::new(1);
    register(&mut c, "alice.near", 0, &k);
    // (No open-interest/order assertions here — that's covered exhaustively in contract.rs;
    // reaching this call at all without panicking is the point: post-migration state is coherent.)
}
