//! Subaccount identity for NEAR users (Development.md §6.1).
//! Layout matches the Go backend's SubaccountIdToBytes32: brokerId(6B BE) ‖ addr20 ‖ n(6B BE).
use near_sdk::env;

/// keccak256("near:" ‖ account_id)[12..32]
pub fn addr20(account_id: &str) -> [u8; 20] {
    let mut preimage = Vec::with_capacity(5 + account_id.len());
    preimage.extend_from_slice(b"near:");
    preimage.extend_from_slice(account_id.as_bytes());
    let hash = env::keccak256_array(&preimage);
    let mut out = [0u8; 20];
    out.copy_from_slice(&hash[12..]);
    out
}

const MAX_48: u64 = 1 << 48;

pub fn subaccount_id(broker_id: u64, account_id: &str, subaccount_number: u64) -> [u8; 32] {
    assert!(broker_id < MAX_48, "broker id must be < 2^48");
    assert!(subaccount_number < MAX_48, "subaccount number must be < 2^48");
    let mut out = [0u8; 32];
    out[0..6].copy_from_slice(&broker_id.to_be_bytes()[2..]);
    out[6..26].copy_from_slice(&addr20(account_id));
    out[26..32].copy_from_slice(&subaccount_number.to_be_bytes()[2..]);
    out
}
