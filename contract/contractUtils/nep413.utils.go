package contractUtils

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/mr-tron/base58"
)

// NEP413_TAG is 2^31 + 413, the Borsh u32 prefix that keeps NEP-413 payloads from ever
// being valid NEAR transactions (Development.md N6).
const NEP413_TAG uint32 = 1<<31 + 413

// NEP413Payload is the message a NEAR wallet signs for signMessage.
type NEP413Payload struct {
	Message     string
	Nonce       [32]byte
	Recipient   string
	CallbackUrl *string
}

func borshString(buf []byte, s string) []byte {
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(s)))
	return append(buf, s...)
}

// NEP413Bytes returns borsh(tag) ‖ borsh(payload), the exact bytes that get hashed.
func NEP413Bytes(p NEP413Payload) []byte {
	buf := binary.LittleEndian.AppendUint32(nil, NEP413_TAG)
	buf = borshString(buf, p.Message)
	buf = append(buf, p.Nonce[:]...)
	buf = borshString(buf, p.Recipient)
	if p.CallbackUrl == nil {
		buf = append(buf, 0)
	} else {
		buf = append(buf, 1)
		buf = borshString(buf, *p.CallbackUrl)
	}
	return buf
}

// NEP413Hash is sha256(borsh(tag) ‖ borsh(payload)); this is what the Ed25519 key signs.
func NEP413Hash(p NEP413Payload) [32]byte {
	return sha256.Sum256(NEP413Bytes(p))
}

// ParseNearEd25519PublicKey decodes "ed25519:<base58>" into a raw 32-byte key.
func ParseNearEd25519PublicKey(publicKey string) (ed25519.PublicKey, error) {
	encoded, ok := strings.CutPrefix(publicKey, "ed25519:")
	if !ok {
		return nil, fmt.Errorf("unsupported key type in %q: only ed25519 is accepted", publicKey)
	}
	raw, err := base58.Decode(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid base58 public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid ed25519 public key length %d", len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

// VerifyNEP413 checks a wallet signMessage result. signature is base64, as wallets return it.
// It only proves the key signed the payload; the caller must still confirm on-chain that the
// key is a FullAccess key of the claimed account (view_access_key) and consume the nonce.
func VerifyNEP413(publicKey string, signature string, p NEP413Payload) error {
	pub, err := ParseNearEd25519PublicKey(publicKey)
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid base64 signature: %w", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("invalid ed25519 signature length %d", len(sig))
	}
	hash := NEP413Hash(p)
	if !ed25519.Verify(pub, hash[:], sig) {
		return fmt.Errorf("NEP-413 signature verification failed")
	}
	return nil
}
