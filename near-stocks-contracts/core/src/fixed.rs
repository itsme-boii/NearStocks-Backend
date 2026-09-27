//! x18 fixed-point helpers (behavior-spec §1).
//! * Stored values are i128 x18; every product of two x18 values is x36 and is computed in I256.
//! * Division is Euclidean to match Go's big.Int.Div (remainder always >= 0).
use ethnum::I256;
use near_sdk::env;

pub const E18: i128 = 1_000_000_000_000_000_000;

#[inline]
pub fn wide(v: i128) -> I256 {
    I256::from(v)
}

/// Euclidean division, identical to Go big.Int.Div for every sign combination.
#[inline]
pub fn ediv(a: I256, b: I256) -> I256 {
    if b == I256::ZERO {
        env::panic_str("division by zero");
    }
    a.div_euclid(b)
}

/// Narrow back to i128, panicking (and reverting the whole batch) on overflow.
#[inline]
pub fn narrow(v: I256) -> i128 {
    i128::try_from(v).unwrap_or_else(|_| env::panic_str("i128 overflow"))
}

/// cutils.Divx18 on an x36 value.
#[inline]
pub fn divx18(x36: I256) -> I256 {
    ediv(x36, wide(E18))
}

/// Divx18(a * b) narrowed: the common "x18 * x18 -> x18" step.
#[inline]
pub fn mul_divx18(a: i128, b: i128) -> i128 {
    narrow(divx18(wide(a) * wide(b)))
}

/// (a * b) / c with a wide intermediate and Euclidean division.
#[inline]
pub fn mul_div(a: i128, b: i128, c: i128) -> i128 {
    narrow(ediv(wide(a) * wide(b), wide(c)))
}

#[inline]
pub fn checked_add(a: i128, b: i128) -> i128 {
    a.checked_add(b).unwrap_or_else(|| env::panic_str("i128 overflow"))
}

#[inline]
pub fn checked_sub(a: i128, b: i128) -> i128 {
    a.checked_sub(b).unwrap_or_else(|| env::panic_str("i128 overflow"))
}

#[inline]
pub fn checked_neg(a: i128) -> i128 {
    a.checked_neg().unwrap_or_else(|| env::panic_str("i128 overflow"))
}
