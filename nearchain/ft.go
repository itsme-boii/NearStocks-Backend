package nearchain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// Native USDC on NEAR (Circle), 6 decimals (Development.md N-ledger).
const (
	USDCMainnet  = "17208628f84f5d6ad33f0da3bbbeb27ffcb398eac501a31bd6ad2011e36133a1"
	USDCDecimals = 6
)

// ScaleUSDCToX18 converts a 6-decimal USDC amount to the ledger's x18 units.
func ScaleUSDCToX18(amount *big.Int) *big.Int {
	return new(big.Int).Mul(amount, big.NewInt(1_000_000_000_000))
}

// ScaleX18ToUSDC converts ledger x18 units to 6-decimal USDC, rounding down (never pays out dust
// the user doesn't own). The remainder stays in the user's ledger balance.
func ScaleX18ToUSDC(amountX18 *big.Int) (usdc *big.Int, dustX18 *big.Int) {
	q, r := new(big.Int).QuoRem(amountX18, big.NewInt(1_000_000_000_000), new(big.Int))
	return q, r
}

func parseU128String(b []byte) (*big.Int, error) {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("expected JSON string amount, got %s", string(b))
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok || v.Sign() < 0 {
		return nil, fmt.Errorf("invalid amount %q", s)
	}
	return v, nil
}

func (c *Client) FtBalanceOf(ctx context.Context, token, account string) (*big.Int, error) {
	b, err := c.CallView(ctx, token, "ft_balance_of", map[string]string{"account_id": account})
	if err != nil {
		return nil, err
	}
	return parseU128String(b)
}

// StorageRegistered reports whether account has NEP-145 storage on token.
func (c *Client) StorageRegistered(ctx context.Context, token, account string) (bool, error) {
	b, err := c.CallView(ctx, token, "storage_balance_of", map[string]string{"account_id": account})
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(b)) != "null", nil
}

func (c *Client) StorageMinimum(ctx context.Context, token string) (*big.Int, error) {
	b, err := c.CallView(ctx, token, "storage_balance_bounds", map[string]string{})
	if err != nil {
		return nil, err
	}
	var v struct {
		Min string `json:"min"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	m, ok := new(big.Int).SetString(v.Min, 10)
	if !ok {
		return nil, fmt.Errorf("invalid storage min %q", v.Min)
	}
	return m, nil
}

// ---- deposit verification ----

type ftTransferEvent struct {
	Standard string `json:"standard"`
	Event    string `json:"event"`
	Data     []struct {
		OldOwnerId string `json:"old_owner_id"`
		NewOwnerId string `json:"new_owner_id"`
		Amount     string `json:"amount"`
	} `json:"data"`
}

// ftTransfers extracts NEP-141 ft_transfer events (EVENT_JSON logs, format verified on a live
// mainnet USDC receipt) emitted by `token` in successful receipts.
func ftTransfers(tx *TxResult, token string) []struct {
	From, To string
	Amount   *big.Int
} {
	var out []struct {
		From, To string
		Amount   *big.Int
	}
	for _, r := range tx.ReceiptsOutcome {
		if r.Outcome.ExecutorId != token || !r.Succeeded() {
			continue
		}
		for _, l := range r.Outcome.Logs {
			payload, ok := strings.CutPrefix(l, "EVENT_JSON:")
			if !ok {
				continue
			}
			var ev ftTransferEvent
			if json.Unmarshal([]byte(payload), &ev) != nil || ev.Standard != "nep141" || ev.Event != "ft_transfer" {
				continue
			}
			for _, d := range ev.Data {
				amt, ok := new(big.Int).SetString(d.Amount, 10)
				if !ok || amt.Sign() <= 0 {
					continue
				}
				out = append(out, struct {
					From, To string
					Amount   *big.Int
				}{d.OldOwnerId, d.NewOwnerId, amt})
			}
		}
	}
	return out
}

var ErrNotADeposit = errors.New("transaction is not a valid deposit")

// VerifyDirectDeposit checks that a finalized transaction signed by `sender` moved `token` to
// `treasury` with a plain ft_transfer and returns the net amount received. ft_transfer_call is
// rejected because the receiver-side refund (ft_resolve_transfer) would make the amount unstable.
func VerifyDirectDeposit(tx *TxResult, sender, treasury, token string) (*big.Int, error) {
	if tx.FinalExecutionStatus != "FINAL" {
		return nil, fmt.Errorf("%w: not final (%s)", ErrNotADeposit, tx.FinalExecutionStatus)
	}
	if !tx.Succeeded() {
		return nil, fmt.Errorf("%w: transaction failed", ErrNotADeposit)
	}
	if tx.Transaction.SignerId != sender {
		return nil, fmt.Errorf("%w: signed by %s, expected %s", ErrNotADeposit, tx.Transaction.SignerId, sender)
	}
	if tx.Transaction.ReceiverId != token {
		return nil, fmt.Errorf("%w: sent to %s, expected token %s", ErrNotADeposit, tx.Transaction.ReceiverId, token)
	}
	methods := tx.FunctionCallMethods()
	if len(methods) == 0 {
		return nil, fmt.Errorf("%w: no function call", ErrNotADeposit)
	}
	for _, m := range methods {
		if m != "ft_transfer" && m != "storage_deposit" {
			return nil, fmt.Errorf("%w: method %s not allowed (use ft_transfer)", ErrNotADeposit, m)
		}
	}
	net := new(big.Int)
	for _, t := range ftTransfers(tx, token) {
		switch {
		case t.From == sender && t.To == treasury:
			net.Add(net, t.Amount)
		case t.From == treasury && t.To == sender:
			net.Sub(net, t.Amount)
		}
	}
	if net.Sign() <= 0 {
		return nil, fmt.Errorf("%w: no transfer from %s to %s", ErrNotADeposit, sender, treasury)
	}
	return net, nil
}

// ---- payouts ----

// Payout builds, signs and sends `amount` of `token` from the signer account to `receiver`,
// registering the receiver's storage first when needed (one transaction, so both actions
// succeed or fail together). Callers must serialize payouts per signer key (nonce).
func (c *Client) Payout(ctx context.Context, signer Signer, signerId, token, receiver string, amount *big.Int) (*TxResult, [32]byte, error) {
	var zero [32]byte
	if amount.Sign() <= 0 {
		return nil, zero, fmt.Errorf("payout amount must be positive")
	}
	key, err := c.ViewAccessKey(ctx, signerId, PublicKeyString(signer.PublicKey()))
	if err != nil {
		return nil, zero, fmt.Errorf("treasury access key: %w", err)
	}
	block, err := c.FinalBlockHash(ctx)
	if err != nil {
		return nil, zero, err
	}
	var actions []Action
	registered, err := c.StorageRegistered(ctx, token, receiver)
	if err != nil {
		return nil, zero, err
	}
	if !registered {
		minDeposit, err := c.StorageMinimum(ctx, token)
		if err != nil {
			return nil, zero, err
		}
		args, _ := json.Marshal(map[string]any{"account_id": receiver, "registration_only": true})
		actions = append(actions, FunctionCall{MethodName: "storage_deposit", Args: args, Gas: 10 * TGas, Deposit: minDeposit})
	}
	args, _ := json.Marshal(map[string]string{"receiver_id": receiver, "amount": amount.String()})
	actions = append(actions, FunctionCall{MethodName: "ft_transfer", Args: args, Gas: 20 * TGas, Deposit: big.NewInt(OneYocto)})

	tx := &Transaction{
		SignerId: signerId, PublicKey: signer.PublicKey(), Nonce: key.Nonce + 1,
		ReceiverId: token, BlockHash: block, Actions: actions,
	}
	signed, hash, err := SignTransaction(tx, signer)
	if err != nil {
		return nil, hash, err
	}
	res, err := c.SendTx(ctx, signed)
	return res, hash, err
}
