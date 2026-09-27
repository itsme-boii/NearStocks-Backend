package nearchain

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"math/big"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// fakeKMS behaves like KMS for one Ed25519 key.
type fakeKMS struct {
	priv  ed25519.PrivateKey
	spec  types.KeySpec
	calls []kms.SignInput
	bad   bool
}

func (f *fakeKMS) GetPublicKey(_ context.Context, _ *kms.GetPublicKeyInput, _ ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error) {
	der, _ := x509.MarshalPKIXPublicKey(f.priv.Public())
	return &kms.GetPublicKeyOutput{PublicKey: der, KeySpec: f.spec}, nil
}

func (f *fakeKMS) Sign(_ context.Context, in *kms.SignInput, _ ...func(*kms.Options)) (*kms.SignOutput, error) {
	f.calls = append(f.calls, *in)
	sig := ed25519.Sign(f.priv, in.Message)
	if f.bad {
		sig[0] ^= 1
	}
	return &kms.SignOutput{Signature: sig}, nil
}

func TestKMSSignerSignsNearTransactions(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	f := &fakeKMS{priv: priv, spec: types.KeySpecEccNistEdwards25519}
	s, err := NewKMSSigner(context.Background(), f, "alias/sequencer")
	if err != nil {
		t.Fatal(err)
	}
	tx := &Transaction{SignerId: "sequencer.near", PublicKey: s.PublicKey(), Nonce: 5, ReceiverId: "near-stocks.near",
		Actions: []Action{FunctionCall{MethodName: "submit_transactions", Args: SubmitArgs(0, [][]byte{{0}}, [][]byte{{}}, [][]byte{{}}), Gas: MaxGas, Deposit: new(big.Int)}}}
	signed, hash, err := SignTransaction(tx, s)
	if err != nil {
		t.Fatal(err)
	}
	sig := signed[len(signed)-64:]
	if !ed25519.Verify(priv.Public().(ed25519.PublicKey), hash[:], sig) {
		t.Fatal("signature does not verify")
	}
	in := f.calls[0]
	if in.MessageType != types.MessageTypeRaw || in.SigningAlgorithm != types.SigningAlgorithmSpecEd25519Sha512 || len(in.Message) != 32 {
		t.Fatalf("unexpected KMS request %+v", in)
	}

	f.bad = true
	if _, err := s.Sign(hash); err == nil {
		t.Fatal("a signature that does not verify must be refused")
	}
	f.spec = types.KeySpecEccNistP256
	if _, err := NewKMSSigner(context.Background(), f, "alias/wrong"); err == nil {
		t.Fatal("non-Ed25519 keys must be refused")
	}
}
