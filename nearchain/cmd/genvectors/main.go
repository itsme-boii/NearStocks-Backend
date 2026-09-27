// genvectors writes the cross-language golden vectors (Development.md §13.2, gate G1).
// Go is the reference: it uses go-ethereum's EIP-712 implementation and the production helpers
// in libs/cutils and contract/contractUtils. Rust (near-stocks-contracts) and TypeScript
// (vectors/ts) re-derive every value independently and must match byte for byte.
//
// Usage (from 100exhange-backend): go run ./nearchain/cmd/genvectors -out vectors
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/contract/types"
	"github/eugenix-io/logx-inf-backend/libs/cutils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/mr-tron/base58"
)

func hx(b []byte) string { return hexutil.Encode(b) }

func write(dir, name string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), append(b, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote", filepath.Join(dir, name))
}

// ---------- addr20 ----------

type addrCase struct {
	AccountId        string `json:"accountId"`
	Addr20           string `json:"addr20"`
	BrokerId         uint   `json:"brokerId"`
	SubaccountNumber int    `json:"subaccountNumber"`
	SubaccountId     string `json:"subaccountId"`
	SubaccountBytes  string `json:"subaccountBytes32"`
}

var accounts = []string{
	"alice.near",
	"near-stocks.near",
	"bob.testnet",
	"a1_b-c.sub.near",
	"98793cd91a3f870fb126f66285808c7e094afcfc4eda8a970f6648cdf0dbd6de", // implicit
	"0x5a4a3f0fcd06cd2d3e5f7d18ea4a3c6b2de0ae7c",                       // eth-implicit
}

var invalidAccounts = []string{"", "a", "Alice.near", ".near", "alice..near", "alice.", "a b", "alice.near-"}

func subaccountBytes(broker uint, addr string, n int) (string, [32]byte) {
	id := cutils.CreateSubaccountId(broker, addr, n)
	b, err := cutils.SubaccountIdToBytes32(id)
	if err != nil {
		log.Fatal(err)
	}
	return id, b
}

func genAddr20(dir string) {
	cases := []addrCase{}
	for _, a := range accounts {
		addr, err := cutils.NearAccountToAddr20(a)
		if err != nil {
			log.Fatalf("%s: %v", a, err)
		}
		id, b := subaccountBytes(1, addr, 0)
		cases = append(cases, addrCase{a, addr, 1, 0, id, hx(b[:])})
	}
	for _, a := range invalidAccounts {
		if _, err := cutils.NearAccountToAddr20(a); err == nil {
			log.Fatalf("expected %q to be rejected", a)
		}
	}
	write(dir, "addr20.json", map[string]any{
		"spec":    "addr20 = keccak256(\"near:\" || account_id)[12..32]; subaccount = brokerId(6B BE) || addr20 || n(6B BE)",
		"valid":   cases,
		"invalid": invalidAccounts,
	})
}

// ---------- EIP-712 ----------

func genEIP712(dir string) {
	// Deterministic throwaway secp256k1 session key. Never use outside tests.
	sk, err := crypto.ToECDSA(crypto.Keccak256([]byte("near-stocks vector session key 1")))
	if err != nil {
		log.Fatal(err)
	}
	sessionKey := crypto.PubkeyToAddress(sk.PublicKey)

	addr, _ := cutils.NearAccountToAddr20("alice.near")
	_, subBytes := subaccountBytes(1, addr, 0)

	type domainOut struct {
		Name              string `json:"name"`
		Version           string `json:"version"`
		ChainId           int64  `json:"chainId"`
		ContractAccount   string `json:"contractAccount"`
		VerifyingContract string `json:"verifyingContract"`
		DomainSeparator   string `json:"domainSeparator"`
	}
	type orderCase struct {
		Domain        domainOut         `json:"domain"`
		Message       map[string]string `json:"message"`
		TypeHash      string            `json:"typeHash"`
		StructHash    string            `json:"structHash"`
		Digest        string            `json:"digest"`
		Signer        string            `json:"signer"`
		Signature     string            `json:"signature"`
		SignerPrivKey string            `json:"signerPrivateKeyTestOnly"`
	}

	orderType := []apitypes.Type{
		{Name: "subAccountId", Type: "bytes32"},
		{Name: "priceX18", Type: "int128"},
		{Name: "amount", Type: "int128"},
		{Name: "expiration", Type: "uint64"},
		{Name: "isReduce", Type: "bool"},
		{Name: "sessionKey", Type: "address"},
		{Name: "chainId", Type: "uint256"},
		{Name: "productId", Type: "uint32"},
	}

	price, _ := new(big.Int).SetString("65000000000000000000000", 10) // 65,000e18
	amounts := []string{"-1500000000000000000", "250000000000000000"} // short 1.5, long 0.25

	cases := []orderCase{}
	nets := []struct {
		acct  string
		chain int64
	}{
		{contractUtils.NEAR_STOCKS_MAINNET_ACCOUNT, contractUtils.NEAR_STOCKS_MAINNET_CHAIN_ID},
		{contractUtils.NEAR_STOCKS_TESTNET_ACCOUNT, contractUtils.NEAR_STOCKS_TESTNET_CHAIN_ID},
	}
	for i, net := range nets {
		domain := contractUtils.NearStocksDomain(net.acct, net.chain)
		amount, _ := new(big.Int).SetString(amounts[i], 10)
		// Numbers go in as decimal strings: apitypes' signed-int encoding (math.U256Bytes)
		// mutates *big.Int arguments in place, which corrupts values hashed more than once.
		msg := apitypes.TypedDataMessage{
			"subAccountId": subBytes,
			"priceX18":     price.String(),
			"amount":       amount.String(),
			"expiration":   "1790000000000",
			"isReduce":     i == 1,
			"sessionKey":   sessionKey.Hex(),
			"chainId":      fmt.Sprint(net.chain),
			"productId":    "1",
		}
		td := apitypes.TypedData{
			Types: apitypes.Types{
				"EIP712Domain": contractUtils.EIP712_DOMAIN_TYPE,
				"Order":        orderType,
			},
			PrimaryType: "Order",
			Domain:      domain,
			Message:     msg,
		}
		digest, _, err := apitypes.TypedDataAndHash(td)
		if err != nil {
			log.Fatal(err)
		}
		domSep, err := td.HashStruct("EIP712Domain", td.Domain.Map())
		if err != nil {
			log.Fatal(err)
		}
		structHash, err := td.HashStruct("Order", td.Message)
		if err != nil {
			log.Fatal(err)
		}
		sig, err := crypto.Sign(digest, sk)
		if err != nil {
			log.Fatal(err)
		}
		sig[64] += 27 // wallet convention; contract normalises v >= 27

		cases = append(cases, orderCase{
			Domain: domainOut{
				Name: domain.Name, Version: domain.Version, ChainId: net.chain,
				ContractAccount:   net.acct,
				VerifyingContract: domain.VerifyingContract,
				DomainSeparator:   hx(domSep),
			},
			Message: map[string]string{
				"subAccountId": hx(subBytes[:]),
				"priceX18":     price.String(),
				"amount":       amount.String(),
				"expiration":   "1790000000000",
				"isReduce":     fmt.Sprint(i == 1),
				"sessionKey":   sessionKey.Hex(),
				"chainId":      fmt.Sprint(net.chain),
				"productId":    "1",
			},
			TypeHash:      hx(td.TypeHash("Order")),
			StructHash:    hx(structHash),
			Digest:        hx(digest),
			Signer:        sessionKey.Hex(),
			Signature:     hx(sig),
			SignerPrivKey: hx(crypto.FromECDSA(sk)),
		})
	}
	write(dir, "eip712.json", map[string]any{
		"spec":  "Development.md §5.5; Order type from C11 (signing.ts / offChainExchange ABI)",
		"order": cases,
	})
}

// ---------- NEP-413 ----------

func genNEP413(dir string) {
	// Deterministic throwaway ed25519 key. Never use outside tests.
	seed := sha256.Sum256([]byte("near-stocks vector nep413 key 1"))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)
	nearPub := "ed25519:" + base58.Encode(pub)

	cb := "https://near-stocks.app/auth/callback"
	var nonce1, nonce2 [32]byte
	copy(nonce1[:], crypto.Keccak256([]byte("nonce-1")))
	copy(nonce2[:], crypto.Keccak256([]byte("nonce-2")))

	payloads := []contractUtils.NEP413Payload{
		{
			Message:   `{"action":"login","sessionKey":"0x5d5A2F2eAc8a2C0dC40b8f0b1d2e8B4E3dFa7d71","expiry":1790518400000,"ts":1790000000000}`,
			Nonce:     nonce1,
			Recipient: contractUtils.NEAR_STOCKS_MAINNET_ACCOUNT,
		},
		{
			Message:     "near-stocks: sign in — ünïcødé ✓",
			Nonce:       nonce2,
			Recipient:   contractUtils.NEAR_STOCKS_TESTNET_ACCOUNT,
			CallbackUrl: &cb,
		},
	}

	type nepCase struct {
		Message     string  `json:"message"`
		Nonce       string  `json:"nonce"`
		NonceBase64 string  `json:"nonceBase64"`
		Recipient   string  `json:"recipient"`
		CallbackUrl *string `json:"callbackUrl"`
		BorshHex    string  `json:"borshHex"`
		Hash        string  `json:"sha256"`
		Signature   string  `json:"signatureBase64"`
	}
	cases := []nepCase{}
	for _, p := range payloads {
		h := contractUtils.NEP413Hash(p)
		sig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, h[:]))
		if err := contractUtils.VerifyNEP413(nearPub, sig, p); err != nil {
			log.Fatal(err)
		}
		// negative check: a different recipient must fail
		tampered := p
		tampered.Recipient = "evil.near"
		if contractUtils.VerifyNEP413(nearPub, sig, tampered) == nil {
			log.Fatal("tampered NEP-413 payload verified")
		}
		cases = append(cases, nepCase{
			Message: p.Message, Nonce: hx(p.Nonce[:]), NonceBase64: base64.StdEncoding.EncodeToString(p.Nonce[:]),
			Recipient: p.Recipient, CallbackUrl: p.CallbackUrl,
			BorshHex: hx(contractUtils.NEP413Bytes(p)), Hash: hx(h[:]), Signature: sig,
		})
	}
	write(dir, "nep413.json", map[string]any{
		"spec":                "sha256(borsh(u32 2^31+413) || borsh({message,nonce:[u8;32],recipient,callbackUrl:Option<String>})), Ed25519",
		"tag":                 contractUtils.NEP413_TAG,
		"publicKey":           nearPub,
		"privateSeedTestOnly": hx(seed[:]),
		"cases":               cases,
	})
}

// ---------- Borsh (provisional order payload) ----------

func le128(v *big.Int) []byte {
	// two's-complement little-endian i128
	m := new(big.Int).Lsh(big.NewInt(1), 128)
	u := new(big.Int).Set(v)
	if u.Sign() < 0 {
		u.Add(u, m)
	}
	be := u.FillBytes(make([]byte, 16))
	for i, j := 0, 15; i < j; i, j = i+1, j-1 {
		be[i], be[j] = be[j], be[i]
	}
	return be
}

func genBorsh(dir string) {
	addr, _ := cutils.NearAccountToAddr20("alice.near")
	_, sub := subaccountBytes(1, addr, 0)
	price, _ := new(big.Int).SetString("65000000000000000000000", 10)
	amount, _ := new(big.Int).SetString("-1500000000000000000", 10)
	sessionKey := common.HexToAddress("0x5d5A2F2eAc8a2C0dC40b8f0b1d2e8B4E3dFa7d71")

	buf := append([]byte{}, sub[:]...)
	buf = append(buf, le128(price)...)
	buf = append(buf, le128(amount)...)
	buf = binary.LittleEndian.AppendUint64(buf, 1790000000000)
	buf = append(buf, 0) // isReduce=false
	buf = append(buf, sessionKey.Bytes()...)
	buf = binary.LittleEndian.AppendUint32(buf, 1)

	envelope := append([]byte{types.MATCH_ORDERS_TXN}, buf...) // type byte || borsh(struct)

	write(dir, "borsh.json", map[string]any{
		"status": "PROVISIONAL — struct layouts are fixed by the G0 behaviour spec; this vector pins the encoding rules",
		"rules":  "u8 type prefix || borsh(struct); ints little-endian; i128 two's complement; bool 1 byte; [u8;N] raw",
		"order": map[string]any{
			"fields":      "subaccount:[u8;32], price_x18:i128, amount:i128, expiration:u64, is_reduce:bool, session_key:[u8;20], product_id:u32",
			"subaccount":  hx(sub[:]),
			"priceX18":    price.String(),
			"amount":      amount.String(),
			"expiration":  "1790000000000",
			"isReduce":    false,
			"sessionKey":  hex.EncodeToString(sessionKey.Bytes()),
			"productId":   1,
			"borshHex":    hx(buf),
			"typeByte":    types.MATCH_ORDERS_TXN,
			"envelopeHex": hx(envelope),
		},
	})
}

func main() {
	out := flag.String("out", "vectors", "output directory")
	flag.Parse()
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	genAddr20(*out)
	genEIP712(*out)
	genNEP413(*out)
	genBorsh(*out)
	genMath(*out)
	genNearTx(*out)
	genNearSign(*out)
	genPayloads(*out)
	genRequests(*out)
}
