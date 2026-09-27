package tests

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"

	"github.com/stretchr/testify/require"
)

// NearOrderDigest must equal the digest pinned in vectors/eip712.json (Rust and TS agree on it).
func TestNearOrderDigestMatchesVectors(t *testing.T) {
	raw, err := os.ReadFile("../../vectors/eip712.json")
	require.NoError(t, err)
	var v struct {
		Order []struct {
			Domain struct {
				ChainId         int64  `json:"chainId"`
				ContractAccount string `json:"contractAccount"`
			} `json:"domain"`
			Message map[string]string `json:"message"`
			Digest  string            `json:"digest"`
		} `json:"order"`
	}
	require.NoError(t, json.Unmarshal(raw, &v))
	require.NotEmpty(t, v.Order)
	for _, c := range v.Order {
		m := c.Message
		var sub [32]byte
		b, _ := hex.DecodeString(m["subAccountId"][2:])
		copy(sub[:], b)
		price, _ := new(big.Int).SetString(m["priceX18"], 10)
		amount, _ := new(big.Int).SetString(m["amount"], 10)
		exp, _ := new(big.Int).SetString(m["expiration"], 10)
		pid, _ := new(big.Int).SetString(m["productId"], 10)
		d, err := contractUtils.NearOrderDigest(c.Domain.ContractAccount, c.Domain.ChainId, sub, price, amount, exp.Uint64(), m["isReduce"] == "true", m["sessionKey"], uint32(pid.Uint64()))
		require.NoError(t, err)
		require.Equal(t, c.Digest, "0x"+hex.EncodeToString(d))
	}
}
