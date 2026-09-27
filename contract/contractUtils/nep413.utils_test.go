package contractUtils

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/mr-tron/base58"
)

func TestVerifyNEP413(t *testing.T) {
	seed := sha256.Sum256([]byte("unit test key"))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := "ed25519:" + base58.Encode(priv.Public().(ed25519.PublicKey))
	p := NEP413Payload{Message: "login", Recipient: NEAR_STOCKS_MAINNET_ACCOUNT}
	h := NEP413Hash(p)
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, h[:]))

	if err := VerifyNEP413(pub, sig, p); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	cb := "https://x"
	for name, tc := range map[string]struct {
		pub, sig string
		p        NEP413Payload
	}{
		"secp256k1 key":     {"secp256k1:" + base58.Encode(make([]byte, 64)), sig, p},
		"short key":         {"ed25519:" + base58.Encode(make([]byte, 31)), sig, p},
		"bad base64":        {pub, "!!!", p},
		"short signature":   {pub, base64.StdEncoding.EncodeToString(make([]byte, 63)), p},
		"other recipient":   {pub, sig, NEP413Payload{Message: "login", Recipient: "evil.near"}},
		"added callbackUrl": {pub, sig, NEP413Payload{Message: "login", Recipient: NEAR_STOCKS_MAINNET_ACCOUNT, CallbackUrl: &cb}},
	} {
		if VerifyNEP413(tc.pub, tc.sig, tc.p) == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestNEP413Tag(t *testing.T) {
	if NEP413_TAG != 2147484061 {
		t.Fatalf("tag = %d", NEP413_TAG)
	}
}
