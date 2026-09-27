# near-stocks golden vectors (gate G1)

Shared test data that pins every cross-language encoding (Development.md §13.2).
Go generates it; Rust and TypeScript re-derive each value with independent code.

| File | What it pins | Checked by |
|---|---|---|
| `addr20.json` | `keccak256("near:" ‖ account_id)[12..32]` and the 32-byte subaccount layout; invalid account IDs | Go, Rust, TS |
| `eip712.json` | near-stocks domain (name `near-stocks`, version `1`, chainId 397 mainnet / 398 testnet, verifyingContract = keccak256(account)[12..]), Order type hash, struct hash, digest, signature and ecrecover | Go (go-ethereum), Rust (hand-written + `env::ecrecover`), TS (ethers v5) |
| `nep413.json` | NEP-413 Borsh bytes, SHA-256 and Ed25519 signature, plus a tampered-recipient rejection | Go, TS (`borsh` + `@near-js/crypto`) |
| `borsh.json` | Borsh encoding rules and the `type byte ‖ borsh(struct)` envelope. **Provisional** until the G0 behavior spec fixes the struct layouts | Go, Rust (`borsh` derive), TS (`borsh`) |

| `math.json` | 1,040 cases produced by calling the production Go code: `PerpBalance.UpdateBalance` (random + multi-step sequences), engine match deltas, `DeductTradingFee` for brokers 1 and 2. Division is Euclidean | Go (source), Rust (`core/tests/math.rs`) |

Run everything: `./verify-all.sh`

Keys in these files are deterministic throwaway test keys. Never use them anywhere else.

## parity.json (gate G3, core)
24 random trading sessions (8 high-volatility) run through the **production Go ledger code** by
`100exhange-backend/services/balance-server/tests/parity_gen_test.go`: matches
(`UpdateLocalBalanceForOrderMatch`), liquidations (`FinaliseLiquidation` on miniredis), settlement
(`SettleLiqPnlUsingSpots`), insurance (`SettleUsingInsurance`), withdrawable checks
(`GetWithdrawableBalance`), funding ticks and open interest. Each step records whether Go accepted
it; the file ends with every subaccount's final balances, the D-7 fee subaccount, open interest and
cumulative funding. `near-stocks-contracts/core/tests/parity.rs` replays every step through
`submit_transactions` and requires identical decisions and identical final state.

Regenerate: `cd services/balance-server && PARITY_OUT=../../vectors/parity.json go test ./tests/ -run TestGenerateParity` (from the repo root)
