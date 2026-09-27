# near-stocks behavior spec (gate G0)

**Status:** draft 2, 2026-09-26 (Phase 3: the core contract implements §2 types 0, 1, 3, 5, 19, 20 and 21 and §3.1–§3.8, with G3 core parity). The spec is reconstructed from code, because the Solidity source isn't available.
**Scope:** every transaction type 0–27, the math the contract must reproduce, and the decisions still open.
**Rule:** anything marked 🔎 was read directly from the cited code. Anything marked ⚠ is a hazard or open decision that must be closed before Phase 3 code depends on it.

Sources (all paths relative to `100exhange-backend/`):
- **[E]** `contract/endpoint.contract.go`: how each transaction is encoded today
- **[V]** `contract/contractUtils/signatureVerifier.utils.go`: EIP-712 types
- **[B]** `libs/subaccountTypes/balance.subaccountTypes.go`: position, PnL and fee math
- **[S]** `libs/subaccount/subaccount.go`: equity, margin, withdrawable balance
- **[BS]** `services/balance-server/services/balance.service.go`
- **[L]** `services/balance-server/services/liquidation.service.go`
- **[F]** `services/cron-server/funding/funding.go`
- **[O]** `services/api-server/controller/options.controller.go`
- **[P]** `pre-markets.controller.go`; **[Y]** `synthetic-spots.controller.go`
- **[K]** `contract/contractUtils/constants.utils.go`

---

## 1. Conventions

| Topic | Rule | Evidence |
|---|---|---|
| Product IDs | **Even = spot, odd = perp.** `MustGetPerpBalance` panics on even IDs, `MustGetSpotBalance` on odd ones | 🔎 [B]:133-164 |
| Quote token | `QUOTE_TOKEN_PRODUCT_ID = 4`. PnL, fees and funding all settle into product 4 | 🔎 [K]:10 |
| Fixed point | Stored values are **x18** (`amount`, `vQuoteBalance`, `lastCumFundingRate`, spot `balance` and `locked`). Products of two x18 values are **x36** | 🔎 [B], [S] |
| ⚠ Integer width | Stored x18 values fit in `i128` (max about 1.7e38, which is 1.7e20 whole units). **x36 intermediates do not.** 1 BTC × $65,000 is already 6.5e40. The contract must do every multiply-then-divide in **256-bit** math (for example `primitive-types`, or a hand-written `I256`) and narrow back to `i128` with a checked conversion | computed |
| ⚠ Division rounding | Go `big.Int.Div` is **Euclidean**: the remainder is always ≥ 0, so a negative dividend rounds toward −∞. Rust's `/` on integers **truncates toward 0**. Every `Div` / `Divx18` must be ported as **Euclidean division** (`div_euclid` semantics on the 256-bit type). This changes results for every negative PnL | 🔎 `libs/cutils/bigNum.utils.go:165`, `libs/ctypes/customdb.type.go:99` |
| AMM account | `AMM_SUBACCOUNT_ID = 0x…0001000000000000000000000000000000000000000001000000000001`. Pays no trading fee, and its health isn't checked on matches | 🔎 [K]:127, [B]:94-103, [BS]:1103 |
| Insurance account | `INSURANCE_SUBACCOUNT_ID = 0x…0005000000000001`, holding the insurance fund's quote balance | 🔎 [K]:130 |
| Envelope | Every transaction is `u8 type ‖ payload`. On NEAR the payload is Borsh. Today's "dynamic" ABI variant (with a 32-byte offset `0x20`) goes away | 🔎 `contractUtils/common.utils.go`, [E]:793-801 |
| Session-key chainId | Today it's hard-coded to **1** (`SESSION_KEY_CHAIN_ID`) in both the domain and the message. On NEAR it's **397** (398 on testnet) | 🔎 [K]:9, `order.controller.go:298` |

---

## 2. Transaction table

"Payload" is today's ABI field list, in order, which becomes the Borsh struct field order. Types map like this: `bytes32`→`[u8;32]`, `address`→`[u8;20]` (session key) or `AccountId` (receiver), `int128`→`i128`, `uint128`→`u128`, `uint256`→`u128` (chain ID and order ID, range-checked), `uint32`→`u32`, `int256`→`I256`.

| ID | Name | NEAR | Signed by | Payload today 🔎 | On-chain effect | Caller |
|---|---|---|---|---|---|---|
| 0 | PERPTICK | keep | sequencer | `uint128 time, int128[] ratesPerSecond` (one per active perp, ordered by `market_tables.id ASC`) | For each perp `i`: `cum[i] += rate[i] × (time − last_time[i])`; then `last_time[i] = time` (§3.3) | [F]:186 |
| 1 | LIQUIDATE_SUBACCOUNT | keep | sequencer and the liquidator's session key | `uint32 productId, int128[] perpPricesX18, int128[] spotPricesX18, Order liquidator, {bytes32 liquidatee, int128 amount}, int128 matchedAmountX18` (the last field only when `UPGRADED_CONTRACT=1`) | §3.6. Requires the liquidatee to be below maintenance margin on-chain | `order.service.go:645` |
| 2 | FINALISE_DEPOSIT | **drop**, replaced by `ft_on_transfer` | — | `bytes32, uint32, int128, uint256 srcChain` | — | — |
| 3 | WITHDRAW_COLLATERAL | keep | session key | `bytes32 sub, address sessionKey, uint32 productId, uint128 amount, uint128 nonce, uint256 destChainId, address receiver, uint256 chainId` | Check the nonce; `amount ≤ withdrawable` (§3.5); debit `amount`; take the flat fee `WITHDRAWAL_FEE_MAP[pid]` (for example 0.5 USDC on product 4); `ft_transfer(receiver, amount − fee)` plus the re-credit callback (Development.md §5.8). On NEAR, `receiver` becomes an `AccountId` and `destChainId` is dropped (cross-chain goes through 1Click off-chain) | `token.controller.go:1388`, fee `:1176` |
| 4, 6 | DUMMY_PLACEHOLDER | reserved | — | — | panics if used | — |
| 5 | MATCH_ORDERS | keep | taker and maker session keys | `uint32 productId, Order taker, Order maker, int128 matchedAmountX18` | §3.2 | `order.service.go:683` |
| 7 | MANUAL_ASSERT | **owner method, not a batch transaction** | owner | no Go builder | an assertion or view helper | none 🔎 |
| 8 | UPDATE_PRODUCT | **owner method** | owner | no Go builder; ABI shape `UpdateProductTx` (spot or perp) | changes product config | none 🔎 |
| 9 | UPDATE_FEE_RATES | **owner method** | owner | no Go builder | changes the fee factor | none 🔎 |
| 10 | SHIFT_BALANCE | ⚠ **D-3** | sequencer | `bytes32 sub` | moves the whole positive balance of product 4 to product 74 (a broker-2 migration) | `token.controller.go:1469` |
| 11, 12 | SHIFT/BURN_BALANCE_KROMA | **drop** | — | `bytes32 sub` | Kroma chain cleanup | — |
| 13 | WITHDRAW_LOGX | keep | session key | `bytes32, int128 amount, address sessionKey, uint256 destChain, address bridgeOut, uint128 nonce, address receiver, uint256 chainId` | debit LogX; `ft_transfer` on `logx-token` plus a callback. `bridgeOut` and `destChain` are dropped | `token.controller.go:1023` |
| 14 | CLAIM_REWARDS | keep | session key | `bytes32, address sessionKey, address staker, uint32 pid, uint128 nonce, uint256 chainId, int256 transientEarningsX18` | Credits the rewards the sequencer computed. ⚠ The contract trusts `transientEarningsX18` (D-6) | `token.controller.go:1158` |
| 15 | STAKE_LOGX | keep | session key | `bytes32, uint32 pid, int128 amount, address staker, address sessionKey, uint128 nonce, uint256 chainId, int256 transientEarningsX18` | debit LogX, then call `staker.stake` plus a callback | `token.controller.go:688` |
| 16 | UNSTAKE_LOGX | keep | session key | same fields as 15 | `staker.unstake` plus a callback that credits LogX | `token.controller.go:817` |
| 17 | REGISTER | **drop**, replaced by a direct `register_session_key` call | — | — | — | — |
| 18 | CLAIM_LOGX | keep | session key | `bytes32, int128 tokenAmount, address sessionKey, uint128 nonce, uint256 chainId` | credits a LogX airdrop claim. ⚠ The allocation proof lives off-chain today (D-6) | `token.controller.go:311` |
| 19 | SETTLE_USER_PNL | keep | sequencer | `bytes32[] subs, uint32[] spotPids, int128[] spotPricesX18` | For each sub with quote < 0: §3.7 | `settle_pnl.go:72` |
| 20 | SOCIALISE_SUBACCOUNT | keep | sequencer | `bytes32 sub, uint32[] spotPids, int128[] spotPricesX18` | Settle using spots, then insurance (§3.8) | `insurance.controller.go:114` |
| 21 | SET_NONCE | keep (low priority) | sequencer | `bytes32 sub, int128 delta` | `nonce += delta`, delta > 0 | only a commented-out caller 🔎 |
| 22 | REWARD_RATE_TICK | keep | sequencer | `address staker, uint256 cumulativeRate` | stores the staking reward rate | `earning.go:156` |
| 23 | CLAIM_CAMPAIGN_REWARDS | ⚠ **D-4** | session key | `bytes32, int128 reward, uint32 pid, uint32 campaignId, address sessionKey, uint128 nonce, uint256 chainId` | credits a campaign reward | **both callers are commented out** 🔎 `token.controller.go:500,1745` |
| 24 | PLACE_OPTIONS_BET | keep | session key | `bytes32, uint32 pid, int128 amount, uint32 intervalMin, uint128 nonce, address sessionKey, uint256 chainId, uint256 orderId, int128 entryPriceX18, uint32 payoutPct, uint32 feePct` | §3.9 | [O]:353 |
| 25 | CLOSE_OPTIONS_BET | keep | sequencer | `uint256 orderId, int128 exitPriceX18` | §3.9 | [O]:517 |
| 26 | PRE_MARKET_ORDER_REQUEST | keep | session key | `bytes32, uint32 pid, int128 amount, bool isBuy, uint128 nonce, address sessionKey, uint256 chainId, int128 quoteDelta, int128 fees` | §3.10 | [P]:590 |
| 27 | SYN_SPOT_ORDER_REQUEST | keep | session key | same fields as 26 | §3.10, using the synthetic-spot ledger | [Y]:471 |

**Order struct** 🔎 [E]:111-119: `{bytes32 subAccountId, int128 priceX18, int128 amount, uint64 expiration, bool isReduce, address sessionKey, uint256 chainId}`. The signed EIP-712 `Order` adds `uint32 productId`, taken from the enclosing match 🔎 [V]:836-845. This is pinned in `vectors/eip712.json`.

---

## 3. Math the contract must reproduce exactly

All division below is **Euclidean** (§1), and all x36 intermediates are **256-bit**.

### 3.1 Perp position update `UpdateBalance(dA, dQ, cumNow)` 🔎 [B]:377-419
```
fee_f  = Divx18(-( (cumNow - last) * vQuote ))   // RealiseFundingFee
last   = cumNow
pnl    = -fee_f
if sign(A) * sign(dA) >= 0:                      // same direction, or either side is zero
    A += dA ; vQuote += dQ ; return (pnl, fee_f)
p1     = min(|A|, |dA|) * sign(dA)
dQ1    = (dQ * p1) / dA                          // Euclidean
dQ2    = dQ - dQ1
rem    = -((vQuote * p1) / A)                    // Euclidean
pnl   += dQ1 + rem
A     += dA
vQuote += dQ2 - rem
return (pnl, fee_f)
```
The caller then adds `pnl` to the spot quote (product 4) 🔎 [B]:125-131.

### 3.2 MATCH_ORDERS 🔎 [E]:810-916, [BS]:1090-1228
1. **Signs:** the taker's amount is negative if the taker sells. Otherwise the maker's amount is negative. `matchedAmount` takes the **maker's** sign 🔎 [E]:828-838.
2. **Signatures:** verify both EIP-712 `Order`s (with `productId`), check that each key is registered for its subaccount and not expired, and check `expiration` against block time.
3. **Update each side** (maker first, then taker, as today):
   - Price `p` = **the maker's price** 🔎 `services/engine/placeOrder.engine.go:226`.
   - `dA_maker = ±m` (maker side), `dA_taker = ±m` (taker side), where `m` = matched amount.
   - `dQ_maker = Divx18(dA_taker × p)` and `dQ_taker = Divx18(dA_maker × p)` 🔎 `placeOrder.engine.go:381-382`. ⚠ Each side is computed **from the other side's signed amount**, not as `−Divx18(dA × p)`. With Euclidean division these differ by 1 wei when `m × p` isn't a multiple of 1e18. **The contract must use exactly this form.** (Closes R-1.)
   - `UpdatePerpBalance(pid, dA, dQ, cum[pid])` for each side.
4. **Fee:** `DeductTradingFee(dQ, isTaker)`, applied after step 3 for each side: `fee = Divx18(|dQ| × factor × 1e13)`, where `factor = 60` if brokerId == 2, **else 0**. Maker and taker use the same rate. The AMM pays nothing 🔎 [B]:100-116, `libs/cutils/common.utils.go:135`. ⚠ **D-1**.
5. **Health:** after the update, the trade is accepted if `isAMM || safety ≥ 0 || safety ≥ safetyBefore`. Here `safety = Divx18(equity − locked − DivxCust(IM × 96, 2))`, so **0.96 × IM** gives a 4% buffer 🔎 [BS]:772-783. A trade that reduces risk is allowed even when the account is underwater. (Closes R-2.)
6. **Open interest:** update long and short OI for non-AMM sides 🔎 [BS]:1177-1189 (`perputils.AddLongShortOIAtReddis`).

### 3.3 Funding 🔎 [F]:332-376
`cum += ratePerSecond × (t − t_last)`. Rates are `int64` per second, x18.
⚠ **H-1:** if a market's rate calculation fails, the cron still sends the previous rate in PERPTICK but **doesn't** update the Redis cumulative ([F]:153-167 vs `storeFundingRate`). The chain and the DB drift apart. **Fix before shadow mode:** always apply the same rate to Redis that is sent on-chain.
⚠ **H-2:** the PERPTICK array is ordered by the DB's active markets (`id ASC`), while liquidation prices use `ALL_PERPS_ON_CONTRACT`. The contract needs an explicit `Vec<(u32 pid, i128 rate)>` so the two lists can never be confused.

### 3.4 Equity and margin 🔎 [S]:157-228
- `spot_x36 = Σ balance × price` over spots with `PRODUCT_MARKET_WEIGHTS[pid] ≠ 0` (weight treated as 1 — see the FIXME in [S]:229)
- `upnl_x36 = Σ (A × price + vQuote × 1e18)`
- `funding_x36 = Σ −vQuote × (cumNow − last)`
- `equity = spot + upnl − funding`
- `IM = Σ Divx18(|A × price| × IMF)` and `MM = Σ Divx18(|A × price| × MMF)`
- `locked_x36 = Σ max(0, locked) × price`
- Health 🔎 [BS]:1232-1251: `belowIM = IM ≠ 0 && equity − locked < IM`; `belowMM = MM ≠ 0 && equity − locked < MM`; `needsInsurance = IM == 0 && equity − locked < 0`

### 3.5 Withdrawable balance 🔎 [S]:285-327
Start from `avail = equity − IM − locked`. Walk the collateral spots in **reverse** `ALL_COLLATERAL_SPOTS` order. For each spot, allow the whole balance if it fits, otherwise `avail / price` (Euclidean), and subtract the allowed value from `avail`.

### 3.6 Liquidation 🔎 [L]:112-266
- **Liquidatee:**
  - `notional = Divx18(amount × matchPrice)`
  - `UpdateBalance(amount, −notional)`
  - `liqFee = Divx18(|notional| × liqFrac)`, where `liqFrac` defaults to 0.015 and is 0.05 for the listed memecoins ([K]:350)
  - `tradeFee = Divx18(|notional| × factor × 1e13)`
  - `pnl_x36 = (realised − liqFee − tradeFee) × 1e18 + quote × 1e18`, then set quote to 0 and run `SettleLiqPnlUsingSpots` (§3.7)
- **Liquidator:** the mirror-image `UpdateBalance`, with no fees. Realised PnL goes into the quote balance.
- ⚠ **H-3:** the liquidation fee is computed but **no code credits it to anyone.** It just disappears from the liquidatee. Decide who receives it (the insurance fund?) — **D-2**.

### 3.7 Settle a negative quote using spots 🔎 [L]:56-94, [BS]:1253-1305
Go through `ALL_SPOTS_ON_CONTRACT` in order while `pnl < 0`, skipping spots with weight 0, balance ≤ 0 or no price:
- if `value > |pnl|`: deduct `(−pnl) / price` (Euclidean), and `pnl = 0`
- otherwise: `pnl += value` and delete the spot

At the end, `quote += Divx18(pnl)`. SETTLE_USER_PNL first zeroes the negative quote and passes it in as `pnl_x36`.

### 3.8 Socialise / insurance 🔎 [L]:268-329
Settle using spots. If the remaining quote is still negative, move it to the insurance fund's quote balance and set the user's quote to 0. **If insurance would go negative, the whole operation fails** (it returns an error, so on-chain it panics).

### 3.9 Options 🔎 [O]:300-520
- **Place:**
  - ⚠ **Corrected in Phase 4:** the signed `amount` is a **base** amount. The stake is `q = Divx18(amount × entryPrice)` (Euclidean), 🔎 `options.controller.go:164`; the sign of `q` gives the direction (> 0 = up) and `|q|` is the quote staked
  - user quote −= |q|; `OPTIONS_X` quote += |q|
  - store `{orderId, sub, pid, q, entryPrice, payoutPct, feePct}`
- **Close** `(orderId, exitPrice)`:
  - `win = q > 0 ? exit > entry : exit < entry`. **A tie is a loss** (strict comparison).
  - On a win:
    - `payout = |q| × payoutPct / 100` and `fee = |q| × feePct / 100`
    - `OPTIONS_X −= payout`; `OPTIONS_FEES += fee`; user `+= payout − fee`
  - Closing a bet that's already closed is a no-op.
  - ⚠ `exitPrice` comes from the sequencer's oracle read, so it's trusted (Development.md §5.7 guards apply).

### 3.10 Pre-market (26) and synthetic spot (27) 🔎 [P]:444-583, [Y]:316-461
The counterparty is a pool subaccount: `X` (`PRE_MARKETS_X` or `SYN_SPOTS_X`) plus a `FEES` account.
- **Buy** (`amount` = gross quote):
  - `fee = amount × feeBps / 10000`
  - user quote −= amount; X quote += amount − fee; FEES quote += fee
  - X ledger[pid] −= quoteDelta; user ledger[pid] += quoteDelta
  - Here `quoteDelta` = **base tokens out**.
- **Sell** (`amount` = base tokens):
  - user ledger[pid] −= amount; X ledger[pid] += amount
  - FEES += fee; X quote −= quoteDelta + fee; user quote += quoteDelta
  - Here `quoteDelta` = **net quote out**.
- ⚠ **H-4 (bug):** on a pre-market **sell**, the fee is credited to `FEES` **pre-market ledger[4]** (`UpdatePreMarketBalance`, [P]:~559) instead of its token balance. Synthetic spot does it correctly (`UpdateTokenBalance`, [Y]:443). The contract should implement the correct version, and the migration must move any stray `premarket[FEES][4]` balance.
- ⚠ Trust: the prices (`BuyTokens` / `SellTokens`) are computed off-chain, so the contract trusts `quoteDelta` and `fees`. Add bounds checks (for example `fees ≤ amount × maxBps`) — **D-5**.

---

## 4. Decisions (approved 2026-09-26: recommendations accepted)

| ID | Decision | Status |
|---|---|---|
| D-1 | Trading fees come from an owner-set **per-broker fee table** in contract config (`perp::FeeTable`), capped at 1% (`MAX_FEE_FACTOR`). `FeeTable::legacy()` reproduces today (broker 2 = 0.06%, others 0). ⚠ **The near-stocks broker ID and its fee rate still need a number from the business before launch** | ✅ implemented in Rust; the value is open |
| D-2 | The liquidation fee is credited to the **insurance fund** | ✅ implemented in the current backend (`FinaliseLiquidation`), tested |
| D-3 | **Drop** SHIFT_BALANCE (10) from the NEAR contract. The broker-2 quote move (product 4 → 74) becomes a one-time migration step (Development.md §11) | ✅ decided |
| D-4 | CLAIM_CAMPAIGN_REWARDS (23): **ID reserved, not built** until campaigns return | ✅ decided |
| D-5 | Sequencer-supplied `quoteDelta`, `fees` and `transientEarnings` are bounded by per-product caps in contract config, and every value is emitted in events | ✅ decided, built in Phases 3–4 |
| D-6 | The sequencer stays the authority for LogX airdrop and reward amounts (same trust as today). A Merkle root for CLAIM_LOGX can be added later | ✅ decided |
| D-7 | The contract credits trading fees (matches and the liquidatee's trading fee) and withdrawal fees plus sub-unit dust to an explicit **fee subaccount** instead of letting them vanish from the ledger (today `DeductTradingFee` only subtracts, so revenue is implicit). The reconciler can then check `Σ balances == custody` exactly, instead of only `≤` | ✅ **approved**; built in the contract (plus DAO-only `sweep_fees`) and in Go (`TRADING_FEES_SUBACCOUNT_ID`, lossless Redis accrual queue); parity compares the fee account |

| D-8 | *New:* the AMM, which skips fees and health checks, gets an owner-set **maximum absolute position per market** (0 = no cap). It may always reduce. Same rule in the contract (`amm_max_position_x18`) and in Go (`markets.amm_max_positionx18`, `PerpetualMarket.AmmPositionAllowed`) | ✅ built; values to be set per market before launch |

| D-9 | LogX staking (15, 16), reward claims (14), airdrop claims (18) and the reward-rate tick (22) are ledger transactions in the core contract, matching the Go ledger (stake = LogX → stLogX), instead of a separate `staker` contract with cross-contract callbacks. Amounts stay backend-computed (D-6) and capped (`ClaimLimits`); LogX leaves through the NEAR withdrawal (13 = 3 with product 0) and the `logx-token` NEP-141 | ✅ built in Phase 4 |
| D-10 | System subaccounts (AMM, insurance, LogX rewards pool) are funded by the owner with `ft_transfer_call` msg `{"system": "amm"\|"insurance"\|"rewards"}`; rewards accepts LogX only. Pre-market / synthetic house supply is minted by the owner with `add_pool_supply` | ✅ built (testnet G4) |
| D-11 | CLAIM_REWARDS (14) and CLAIM_LOGX (18) are paid from the DAO-funded LogX rewards pool (`LOGX_REWARDS_SUBACCOUNT`), not minted: every ledger LogX stays backed by tokens the contract holds. A claim the pool cannot cover panics ("LogX rewards pool exhausted"); the backend reserves from the same pool first | ✅ built (testnet G4) |

## 5. Hazards

| ID | Status |
|---|---|
| H-1 funding drift | ✅ **Fixed** in `services/cron-server/funding/funding.go`. The rate sent in PerpTick (fresh or fallback) is always stored in Redis. OI-cap parse failures take the same path. A failed store raises a `FUNDING DRIFT` Discord alert. Tested by `TestSelectFundingRate` |
| H-2 PERPTICK ordering | ✅ **Resolved by design:** the NEAR PERPTICK payload is `Vec<(u32 product_id, i64 rate_per_second)>`, with only markets whose Redis update succeeded. Built in Phase 4 (the Borsh encoder) |
| H-3 liquidation fee never credited | ✅ **Fixed** (D-2): `FinaliseLiquidation` credits `LiquidationFeex18` to `INSURANCE_SUBACCOUNT_ID` in the same Redis transaction. Lock order is users first, insurance last, and there's no self-lock when insurance is a party. Tested by `TestLiquidationFeeCreditedToInsurance`; a mutation check confirms the test fails without the fix |
| H-5 lost updates in `AtomicUpdateBalanceForOrderMatch` | ✅ **Fixed.** The locks were taken inside `TxPipelined`, so they were released before EXEC; concurrent matches on one account (the AMM) could overwrite each other. The locks now wrap the pipeline. `TestConcurrentMatchesAgainstAmmLoseNothing` fails with the old ordering |
| H-6 lost unlocks in `BalanceLockerService.AtomicUpdateSubaccountForMatch` | ✅ **Fixed** the same way. The existing `TestAtomicUpdateSubaccountForMatch` was flaky because of it (4 of 15 runs failed on `HEAD`) and now passes 15 of 15 |
| H-4 pre-market sell fee ledger | ✅ **Fixed** in `pre-markets.controller.go`: `UpdateTokenBalance`, matching the buy path and synthetic spot. **Fees collected before the fix** are still in `premarket[PRE_MARKETS_FEES][4]` and are moved by a one-time migration step (Development.md §11). Nothing reads that ledger |

## 6. Replay items for the parity harness (G3)
- ~~R-1~~ closed (§3.2 step 3). ~~R-2~~ closed (§3.2 step 5).
- **R-4:** the fee recorded on the fill (`order.service.go` `createFillsForMatchedOrders`, based on `Divx18(m × p)`) and the fee actually deducted (`DeductTradingFee`, based on `|dQ|`) can differ by 1 wei on the side with negative `dQ`. The reconciler must compare balances, not fill fees
- ~~R-3~~ closed: `core/tests/invariants.rs` (proptest) plus the G3 replay (`core/tests/parity.rs`, 3,840 Go steps, zero differences).
- ~~R-5~~ **decided and aligned:** a non-AMM liquidator must pass the match rule (`safety ≥ 0 or ≥ before`) at the liquidation's prices, and an AMM liquidator must respect D-8. Go's `FinaliseLiquidation` applies exactly this before writing, so the backend never sends a liquidation the contract would refuse. The parity vectors include refused liquidations.
- **R-6 (inherited rounding):** `SettleLiqPnlUsingSpots` floors `(-pnl)/price` and then sets pnl to 0, so the user keeps less than one wei of the consumed spot per settlement. Kept for parity and pinned by a property test.
- **R-7 (off-chain rule kept off-chain):** the unstake cooldown (`GetUnstakeSumOfPendingUnstake`) lives only in the backend DB. The contract enforces balances; the backend keeps refusing to stake or withdraw LogX that is still cooling down, as it does for open-order locks.
- **On-chain `locked` = 0:** open-order locks exist only in Redis, so every on-chain equity, health and withdrawable computation uses `locked = 0`. For the match rule this is never stricter than the engine (both sides of `safety ≥ before` shift equally; `safety ≥ 0` only gets easier). The backend still enforces locks before it creates a withdrawal.
