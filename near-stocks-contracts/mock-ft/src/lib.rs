//! Minimal NEP-141 for sandbox tests only. Implements what near-stocks touches: storage
//! registration, ft_transfer, ft_transfer_call with ft_resolve_transfer refunds, and balances.
//! `set_fail` makes every ft_transfer to an account panic, to test the withdrawal re-credit path.
use near_sdk::json_types::U128;
use near_sdk::serde_json::{json, Value};
use near_sdk::store::LookupMap;
use near_sdk::{env, near, AccountId, Gas, NearToken, PanicOnDefault, Promise, PromiseError};

#[near(contract_state)]
#[derive(PanicOnDefault)]
pub struct MockFt {
    balances: LookupMap<AccountId, u128>,
    failing: Vec<AccountId>,
}

#[near]
impl MockFt {
    #[init]
    pub fn new() -> Self {
        Self { balances: LookupMap::new(b"b"), failing: vec![] }
    }

    pub fn mint(&mut self, account_id: AccountId, amount: U128) {
        let b = self.balances.get(&account_id).copied().unwrap_or(0);
        self.balances.insert(account_id, b + amount.0);
    }

    pub fn set_fail(&mut self, account_id: AccountId, fail: bool) {
        self.failing.retain(|a| a != &account_id);
        if fail {
            self.failing.push(account_id);
        }
    }

    #[payable]
    pub fn storage_deposit(&mut self, account_id: Option<AccountId>, registration_only: Option<bool>) -> Value {
        let _ = registration_only;
        let a = account_id.unwrap_or_else(env::predecessor_account_id);
        if !self.balances.contains_key(&a) {
            self.balances.insert(a, 0);
        }
        json!({ "total": "1250000000000000000000", "available": "0" })
    }

    pub fn storage_balance_of(&self, account_id: AccountId) -> Option<Value> {
        self.balances.contains_key(&account_id).then(|| json!({ "total": "1250000000000000000000", "available": "0" }))
    }

    pub fn ft_balance_of(&self, account_id: AccountId) -> U128 {
        U128(self.balances.get(&account_id).copied().unwrap_or(0))
    }

    #[payable]
    pub fn ft_transfer(&mut self, receiver_id: AccountId, amount: U128, memo: Option<String>) {
        let _ = memo;
        assert_eq!(env::attached_deposit(), NearToken::from_yoctonear(1), "Requires attached deposit of exactly 1 yoctoNEAR");
        if self.failing.contains(&receiver_id) {
            env::panic_str("forced transfer failure");
        }
        self.move_tokens(&env::predecessor_account_id(), &receiver_id, amount.0);
    }

    #[payable]
    pub fn ft_transfer_call(&mut self, receiver_id: AccountId, amount: U128, memo: Option<String>, msg: String) -> Promise {
        let _ = memo;
        assert_eq!(env::attached_deposit(), NearToken::from_yoctonear(1), "Requires attached deposit of exactly 1 yoctoNEAR");
        let sender = env::predecessor_account_id();
        self.move_tokens(&sender, &receiver_id, amount.0);
        Promise::new(receiver_id.clone())
            .function_call(
                "ft_on_transfer",
                json!({ "sender_id": sender, "amount": amount, "msg": msg }).to_string().into_bytes(),
                NearToken::from_yoctonear(0),
                Gas::from_tgas(40),
            )
            .then(Self::ext(env::current_account_id()).with_static_gas(Gas::from_tgas(10)).ft_resolve_transfer(sender, receiver_id, amount))
    }

    #[private]
    pub fn ft_resolve_transfer(
        &mut self,
        sender_id: AccountId,
        receiver_id: AccountId,
        amount: U128,
        #[callback_result] unused: Result<U128, PromiseError>,
    ) -> U128 {
        let unused = unused.map(|u| u.0.min(amount.0)).unwrap_or(amount.0);
        let refund = unused.min(self.balances.get(&receiver_id).copied().unwrap_or(0));
        if refund > 0 {
            self.move_tokens(&receiver_id, &sender_id, refund);
        }
        U128(amount.0 - refund)
    }
}

impl MockFt {
    fn move_tokens(&mut self, from: &AccountId, to: &AccountId, amount: u128) {
        let fb = self.balances.get(from).copied().unwrap_or_else(|| env::panic_str("sender not registered"));
        let tb = self.balances.get(to).copied().unwrap_or_else(|| env::panic_str("The account is not registered"));
        if fb < amount {
            env::panic_str("The account doesn't have enough balance");
        }
        self.balances.insert(from.clone(), fb - amount);
        self.balances.insert(to.clone(), tb + amount);
    }
}
