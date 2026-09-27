//! Hand-written EIP-712 for session-key signatures (Development.md §5.5, spike S-3).
//! Only hashing follows ABI rules; payloads travel as Borsh.
use near_sdk::borsh::{BorshDeserialize, BorshSerialize};
use near_sdk::env;

pub const DOMAIN_NAME: &str = "near-stocks";
pub const DOMAIN_VERSION: &str = "1";

const DOMAIN_TYPE: &[u8] = b"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)";
const ORDER_TYPE: &[u8] = b"Order(bytes32 subAccountId,int128 priceX18,int128 amount,uint64 expiration,bool isReduce,address sessionKey,uint256 chainId,uint32 productId)";

fn keccak(data: &[u8]) -> [u8; 32] {
    env::keccak256_array(data)
}

// ---- ABI word encoders (each value becomes one 32-byte word) ----

fn word_u128(v: u128) -> [u8; 32] {
    let mut w = [0u8; 32];
    w[16..].copy_from_slice(&v.to_be_bytes());
    w
}

fn word_i128(v: i128) -> [u8; 32] {
    // sign-extend to 256 bits
    let mut w = if v < 0 { [0xffu8; 32] } else { [0u8; 32] };
    w[16..].copy_from_slice(&v.to_be_bytes());
    w
}

fn word_address(a: &[u8; 20]) -> [u8; 32] {
    let mut w = [0u8; 32];
    w[12..].copy_from_slice(a);
    w
}

/// keccak256(contract_account_id)[12..32]: the 20-byte verifyingContract stand-in.
pub fn verifying_contract(contract_account_id: &str) -> [u8; 20] {
    let h = keccak(contract_account_id.as_bytes());
    let mut out = [0u8; 20];
    out.copy_from_slice(&h[12..]);
    out
}

pub fn domain_separator(chain_id: u64, verifying_contract: &[u8; 20]) -> [u8; 32] {
    let mut buf = Vec::with_capacity(32 * 5);
    buf.extend_from_slice(&keccak(DOMAIN_TYPE));
    buf.extend_from_slice(&keccak(DOMAIN_NAME.as_bytes()));
    buf.extend_from_slice(&keccak(DOMAIN_VERSION.as_bytes()));
    buf.extend_from_slice(&word_u128(chain_id as u128));
    buf.extend_from_slice(&word_address(verifying_contract));
    keccak(&buf)
}

/// A user order (C11). Field order matches the EIP-712 type.
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct Order {
    pub subaccount: [u8; 32],
    pub price_x18: i128,
    pub amount: i128,
    pub expiration: u64,
    pub is_reduce: bool,
    pub session_key: [u8; 20],
    pub product_id: u32,
}

pub fn order_type_hash() -> [u8; 32] {
    keccak(ORDER_TYPE)
}

pub fn order_struct_hash(o: &Order, chain_id: u64) -> [u8; 32] {
    let mut buf = Vec::with_capacity(32 * 9);
    buf.extend_from_slice(&order_type_hash());
    buf.extend_from_slice(&o.subaccount);
    buf.extend_from_slice(&word_i128(o.price_x18));
    buf.extend_from_slice(&word_i128(o.amount));
    buf.extend_from_slice(&word_u128(o.expiration as u128));
    buf.extend_from_slice(&word_u128(o.is_reduce as u128));
    buf.extend_from_slice(&word_address(&o.session_key));
    buf.extend_from_slice(&word_u128(chain_id as u128));
    buf.extend_from_slice(&word_u128(o.product_id as u128));
    keccak(&buf)
}

const NEAR_WITHDRAW_TYPE: &[u8] =
    b"NearWithdraw(bytes32 subAccountId,address sessionKey,uint32 productId,uint128 amount,uint128 nonce,string receiver,uint256 chainId)";

/// WITHDRAW_COLLATERAL payload (tx 3). The session key signs the exact NEAR receiver
/// (contractUtils.NearWithdrawType, pinned in vectors/nearsign.json).
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct NearWithdraw {
    pub subaccount: [u8; 32],
    pub session_key: [u8; 20],
    pub product_id: u32,
    pub amount: u128,
    pub nonce: u128,
    pub receiver: String,
}

pub fn near_withdraw_struct_hash(w: &NearWithdraw, chain_id: u64) -> [u8; 32] {
    let mut buf = Vec::with_capacity(32 * 8);
    buf.extend_from_slice(&keccak(NEAR_WITHDRAW_TYPE));
    buf.extend_from_slice(&w.subaccount);
    buf.extend_from_slice(&word_address(&w.session_key));
    buf.extend_from_slice(&word_u128(w.product_id as u128));
    buf.extend_from_slice(&word_u128(w.amount));
    buf.extend_from_slice(&word_u128(w.nonce));
    buf.extend_from_slice(&keccak(w.receiver.as_bytes()));
    buf.extend_from_slice(&word_u128(chain_id as u128));
    keccak(&buf)
}

pub fn typed_digest(domain_separator: &[u8; 32], struct_hash: &[u8; 32]) -> [u8; 32] {
    let mut buf = [0u8; 66];
    buf[0] = 0x19;
    buf[1] = 0x01;
    buf[2..34].copy_from_slice(domain_separator);
    buf[34..].copy_from_slice(struct_hash);
    keccak(&buf)
}

/// Recovers the signer address of a 65-byte r‖s‖v signature. v may be 0/1 or 27/28.
/// `malleability_flag = true` rejects high-s signatures.
pub fn recover_signer(digest: &[u8; 32], signature: &[u8]) -> Option<[u8; 20]> {
    if signature.len() != 65 {
        return None;
    }
    let mut v = signature[64];
    if v >= 27 {
        v -= 27;
    }
    if v > 1 {
        return None;
    }
    let pubkey = env::ecrecover(digest, &signature[..64], v, true)?;
    let h = keccak(&pubkey);
    let mut addr = [0u8; 20];
    addr.copy_from_slice(&h[12..]);
    Some(addr)
}

// ---------------------------------------------------------------------------------------------
// Generic struct hashing for the user-signed request types that are kept from the current system
// (contractUtils/signatureVerifier.utils.go). Field lists are unchanged; only the domain is new.

#[derive(Clone, Copy, Debug)]
pub enum Field<'a> {
    Bytes32(&'a [u8; 32]),
    Address(&'a [u8; 20]),
    Uint(u128),
    Int(i128),
    Bool(bool),
}

pub fn struct_hash(type_string: &[u8], fields: &[Field]) -> [u8; 32] {
    let mut buf = Vec::with_capacity(32 * (fields.len() + 1));
    buf.extend_from_slice(&keccak(type_string));
    for f in fields {
        let w = match *f {
            Field::Bytes32(b) => *b,
            Field::Address(a) => word_address(a),
            Field::Uint(u) => word_u128(u),
            Field::Int(i) => word_i128(i),
            Field::Bool(b) => word_u128(b as u128),
        };
        buf.extend_from_slice(&w);
    }
    keccak(&buf)
}

pub const USER_OPTION_BET_TYPE: &[u8] =
    b"UserOptionBet(bytes32 subAccountId,uint32 productId,int128 amount,uint32 interval,uint128 nonce,address sessionKey,uint256 chainId)";
pub const PRE_MARKET_ORDER_TYPE: &[u8] =
    b"PlacePreMarketOrderRequest(bytes32 subAccountId,uint32 productId,int128 amount,bool isBuy,uint128 nonce,address sessionKey,uint256 chainId)";
pub const SYN_SPOT_ORDER_TYPE: &[u8] =
    b"PlaceSyntheticSpotOrderRequest(bytes32 subAccountId,uint32 productId,int128 amount,bool isBuy,uint128 nonce,address sessionKey,uint256 chainId)";

/// Signed by the user's session key: UserOptionBet (tx 24).
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct OptionBet {
    pub subaccount: [u8; 32],
    pub product_id: u32,
    /// base amount, x18; sign = direction (> 0 up). The stake is Divx18(amount * entry price).
    pub amount: i128,
    pub interval: u32,
    pub nonce: u128,
    pub session_key: [u8; 20],
}

pub fn option_bet_hash(o: &OptionBet, chain_id: u64) -> [u8; 32] {
    use Field::*;
    struct_hash(
        USER_OPTION_BET_TYPE,
        &[
            Bytes32(&o.subaccount),
            Uint(o.product_id as u128),
            Int(o.amount),
            Uint(o.interval as u128),
            Uint(o.nonce),
            Address(&o.session_key),
            Uint(chain_id as u128),
        ],
    )
}

/// Signed by the user: PlacePreMarketOrderRequest (26) or PlaceSyntheticSpotOrderRequest (27).
#[derive(BorshSerialize, BorshDeserialize, Clone, Debug, PartialEq, Eq)]
#[borsh(crate = "near_sdk::borsh")]
pub struct PoolOrder {
    pub subaccount: [u8; 32],
    pub product_id: u32,
    /// buy: gross quote paid; sell: base tokens sold (x18, > 0)
    pub amount: i128,
    pub is_buy: bool,
    pub nonce: u128,
    pub session_key: [u8; 20],
}

pub fn pool_order_hash(type_string: &[u8], o: &PoolOrder, chain_id: u64) -> [u8; 32] {
    use Field::*;
    struct_hash(
        type_string,
        &[
            Bytes32(&o.subaccount),
            Uint(o.product_id as u128),
            Int(o.amount),
            Bool(o.is_buy),
            Uint(o.nonce),
            Address(&o.session_key),
            Uint(chain_id as u128),
        ],
    )
}
