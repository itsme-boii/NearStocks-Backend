package nearchain

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	"github.com/mr-tron/base58"
)

// Borsh layout from nearcore core/primitives/src/transaction.rs (TransactionV0, serialized with
// no version tag) and action/mod.rs (Action::FunctionCall = 2, Action::Transfer = 3).

const (
	keyTypeED25519            = 0
	actionFunctionCall        = 2
	actionTransfer            = 3
	OneYocto                  = 1
	TGas               uint64 = 1_000_000_000_000
)

type Action interface{ encode(buf []byte) []byte }

type FunctionCall struct {
	MethodName string
	Args       []byte
	Gas        uint64
	Deposit    *big.Int // yoctoNEAR, u128
}

type Transfer struct {
	Deposit *big.Int
}

type Transaction struct {
	SignerId   string
	PublicKey  ed25519.PublicKey
	Nonce      uint64
	ReceiverId string
	BlockHash  [32]byte
	Actions    []Action
}

func putString(buf []byte, s string) []byte {
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(s)))
	return append(buf, s...)
}

func putU128(buf []byte, v *big.Int) []byte {
	if v == nil {
		v = new(big.Int)
	}
	if v.Sign() < 0 || v.BitLen() > 128 {
		panic(fmt.Sprintf("u128 out of range: %s", v))
	}
	be := v.FillBytes(make([]byte, 16))
	for i := 15; i >= 0; i-- {
		buf = append(buf, be[i])
	}
	return buf
}

func (f FunctionCall) encode(buf []byte) []byte {
	buf = append(buf, actionFunctionCall)
	buf = putString(buf, f.MethodName)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(f.Args)))
	buf = append(buf, f.Args...)
	buf = binary.LittleEndian.AppendUint64(buf, f.Gas)
	return putU128(buf, f.Deposit)
}

// DeployContract is action 1 (Borsh enum index): the account's new wasm code.
type DeployContract struct{ Code []byte }

func (d DeployContract) encode(buf []byte) []byte {
	buf = append(buf, 1)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(d.Code)))
	return append(buf, d.Code...)
}

func (t Transfer) encode(buf []byte) []byte {
	buf = append(buf, actionTransfer)
	return putU128(buf, t.Deposit)
}

// Encode returns borsh(TransactionV0).
func (tx *Transaction) Encode() []byte {
	buf := putString(nil, tx.SignerId)
	buf = append(buf, keyTypeED25519)
	buf = append(buf, tx.PublicKey...)
	buf = binary.LittleEndian.AppendUint64(buf, tx.Nonce)
	buf = putString(buf, tx.ReceiverId)
	buf = append(buf, tx.BlockHash[:]...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(tx.Actions)))
	for _, a := range tx.Actions {
		buf = a.encode(buf)
	}
	return buf
}

// Hash is sha256(borsh(tx)); it is both what gets signed and the transaction hash.
func (tx *Transaction) Hash() [32]byte {
	return sha256.Sum256(tx.Encode())
}

// Signer produces an Ed25519 signature over a 32-byte transaction hash. The local
// implementation is used for testnet/demo; production swaps in AWS KMS (Development.md §8.2).
type Signer interface {
	PublicKey() ed25519.PublicKey
	Sign(hash [32]byte) ([]byte, error)
}

type LocalSigner struct{ key ed25519.PrivateKey }

// ParseNearPrivateKey accepts "ed25519:<base58>" holding a 64-byte (seed‖pub) or 32-byte seed key.
func ParseNearPrivateKey(s string) (*LocalSigner, error) {
	enc, ok := strings.CutPrefix(strings.TrimSpace(s), "ed25519:")
	if !ok {
		return nil, fmt.Errorf("private key must start with ed25519:")
	}
	raw, err := base58.Decode(enc)
	if err != nil {
		return nil, fmt.Errorf("invalid base58 private key: %w", err)
	}
	switch len(raw) {
	case ed25519.PrivateKeySize:
		k := ed25519.PrivateKey(raw)
		if !k.Public().(ed25519.PublicKey).Equal(ed25519.NewKeyFromSeed(raw[:32]).Public()) {
			return nil, fmt.Errorf("private key halves do not match")
		}
		return &LocalSigner{k}, nil
	case ed25519.SeedSize:
		return &LocalSigner{ed25519.NewKeyFromSeed(raw)}, nil
	default:
		return nil, fmt.Errorf("invalid ed25519 private key length %d", len(raw))
	}
}

func NewLocalSigner(key ed25519.PrivateKey) *LocalSigner { return &LocalSigner{key} }

func (s *LocalSigner) PublicKey() ed25519.PublicKey { return s.key.Public().(ed25519.PublicKey) }

func (s *LocalSigner) Sign(hash [32]byte) ([]byte, error) { return ed25519.Sign(s.key, hash[:]), nil }

// PublicKeyString renders "ed25519:<base58>".
func PublicKeyString(pk ed25519.PublicKey) string { return "ed25519:" + base58.Encode(pk) }

// SignTransaction returns borsh(SignedTransaction) and the transaction hash.
func SignTransaction(tx *Transaction, signer Signer) ([]byte, [32]byte, error) {
	if !tx.PublicKey.Equal(signer.PublicKey()) {
		return nil, [32]byte{}, fmt.Errorf("transaction public key does not match signer")
	}
	hash := tx.Hash()
	sig, err := signer.Sign(hash)
	if err != nil {
		return nil, hash, err
	}
	if len(sig) != ed25519.SignatureSize || !ed25519.Verify(tx.PublicKey, hash[:], sig) {
		return nil, hash, fmt.Errorf("signer returned an invalid signature")
	}
	out := tx.Encode()
	out = append(out, keyTypeED25519)
	out = append(out, sig...)
	return out, hash, nil
}

// HashString renders a transaction hash the way NEAR explorers and RPC expect (base58).
func HashString(h [32]byte) string {
	if h == ([32]byte{}) {
		return ""
	}
	return base58.Encode(h[:])
}
