package nearchain

// AWS KMS Ed25519 signer for the sequencer and relayer keys (Development.md §12, N15): KMS key
// spec ECC_NIST_EDWARDS25519, SigningAlgorithm ED25519_SHA_512, MessageType RAW over the 32-byte
// NEAR transaction hash. The private key never leaves KMS.

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// KMSAPI is the part of the KMS client the signer uses (a fake implements it in tests).
type KMSAPI interface {
	GetPublicKey(ctx context.Context, in *kms.GetPublicKeyInput, opts ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error)
	Sign(ctx context.Context, in *kms.SignInput, opts ...func(*kms.Options)) (*kms.SignOutput, error)
}

type KMSSigner struct {
	client KMSAPI
	keyId  string
	pub    ed25519.PublicKey
}

// NewKMSSigner loads the key's public half and checks it is an Ed25519 key.
func NewKMSSigner(ctx context.Context, client KMSAPI, keyId string) (*KMSSigner, error) {
	out, err := client.GetPublicKey(ctx, &kms.GetPublicKeyInput{KeyId: aws.String(keyId)})
	if err != nil {
		return nil, fmt.Errorf("kms GetPublicKey: %w", err)
	}
	if out.KeySpec != types.KeySpecEccNistEdwards25519 {
		return nil, fmt.Errorf("kms key %s has spec %s, want %s", keyId, out.KeySpec, types.KeySpecEccNistEdwards25519)
	}
	parsed, err := x509.ParsePKIXPublicKey(out.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("kms public key: %w", err)
	}
	pub, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("kms public key is %T, not ed25519", parsed)
	}
	return &KMSSigner{client: client, keyId: keyId, pub: pub}, nil
}

func (s *KMSSigner) PublicKey() ed25519.PublicKey { return s.pub }

// Sign signs a transaction hash in KMS and verifies the result locally before returning it.
func (s *KMSSigner) Sign(hash [32]byte) ([]byte, error) {
	out, err := s.client.Sign(context.Background(), &kms.SignInput{
		KeyId:            aws.String(s.keyId),
		Message:          hash[:],
		MessageType:      types.MessageTypeRaw,
		SigningAlgorithm: types.SigningAlgorithmSpecEd25519Sha512,
	})
	if err != nil {
		return nil, fmt.Errorf("kms Sign: %w", err)
	}
	if len(out.Signature) != ed25519.SignatureSize || !ed25519.Verify(s.pub, hash[:], out.Signature) {
		return nil, fmt.Errorf("kms returned a signature that does not verify")
	}
	return out.Signature, nil
}

// NewKMSSignerFromAWS uses the default AWS credential chain (instance role, env, profile).
func NewKMSSignerFromAWS(ctx context.Context, keyId string) (*KMSSigner, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	return NewKMSSigner(ctx, kms.NewFromConfig(cfg), keyId)
}

// SequencerFromEnv builds the sequencer from NEAR_RPC_URL, NEAR_STOCKS_ACCOUNT,
// NEAR_SEQUENCER_ACCOUNT and either NEAR_SEQUENCER_KMS_KEY_ID (production) or
// NEAR_SEQUENCER_PRIVATE_KEY ("ed25519:...", testnet and local only).
func SequencerFromEnv(ctx context.Context, contractAccount string) (*Sequencer, error) {
	rpcURL := os.Getenv("NEAR_RPC_URL")
	account := os.Getenv("NEAR_SEQUENCER_ACCOUNT")
	if rpcURL == "" || account == "" {
		return nil, fmt.Errorf("NEAR_RPC_URL and NEAR_SEQUENCER_ACCOUNT are required")
	}
	var signer Signer
	if keyId := os.Getenv("NEAR_SEQUENCER_KMS_KEY_ID"); keyId != "" {
		s, err := NewKMSSignerFromAWS(ctx, keyId)
		if err != nil {
			return nil, err
		}
		signer = s
	} else if raw := os.Getenv("NEAR_SEQUENCER_PRIVATE_KEY"); raw != "" {
		s, err := ParseNearPrivateKey(raw)
		if err != nil {
			return nil, err
		}
		signer = s
	} else {
		return nil, fmt.Errorf("set NEAR_SEQUENCER_KMS_KEY_ID or NEAR_SEQUENCER_PRIVATE_KEY")
	}
	return &Sequencer{RPC: NewClient(rpcURL), Signer: signer, SignerId: account, Contract: contractAccount}, nil
}
