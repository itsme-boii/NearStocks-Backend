package main

// near-stocks session-key messages (Register, NearWithdraw): Go backend verifier vs ethers (frontend).

import (
	"log"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"

	"github.com/ethereum/go-ethereum/crypto"
)

func genNearSign(dir string) {
	sk, _ := crypto.ToECDSA(crypto.Keccak256([]byte("near-stocks vector session key 1")))
	sessionKey := crypto.PubkeyToAddress(sk.PublicKey).Hex()
	addr, _ := cutils.NearAccountToAddr20("alice.near")
	sub := cutils.CreateSubaccountId(1, addr, 0)
	subBytes, _ := cutils.SubaccountIdToBytes32(sub)
	acct, chain := contractUtils.NEAR_STOCKS_MAINNET_ACCOUNT, int64(contractUtils.NEAR_STOCKS_MAINNET_CHAIN_ID)

	sign := func(h []byte) string {
		sig, err := crypto.Sign(h, sk)
		if err != nil {
			log.Fatal(err)
		}
		sig[64] += 27
		return hx(sig)
	}

	reg := contractUtils.NearRegister{SubaccountId: sub, UserAddress: addr, SessionKey: sessionKey, ExpiryTs: 1790518400000, Nonce: 0}
	rh, err := contractUtils.NearRegisterHash(acct, chain, reg)
	if err != nil {
		log.Fatal(err)
	}
	rsig := sign(rh)
	if err := contractUtils.VerifyNearRegister(acct, chain, reg, rsig); err != nil {
		log.Fatal(err)
	}

	wd := contractUtils.NearWithdraw{SubaccountId: sub, SessionKey: sessionKey, ProductId: 4, Amount: "25500000000000000000", Nonce: 3, Receiver: "alice.near"}
	wh, err := contractUtils.NearWithdrawHash(acct, chain, wd)
	if err != nil {
		log.Fatal(err)
	}
	wsig := sign(wh)
	if err := contractUtils.VerifyNearWithdraw(acct, chain, wd, wsig); err != nil {
		log.Fatal(err)
	}
	tampered := wd
	tampered.Receiver = "mallory.near"
	if contractUtils.VerifyNearWithdraw(acct, chain, tampered, wsig) == nil {
		log.Fatal("tampered receiver verified")
	}

	write(dir, "nearsign.json", map[string]any{
		"spec":            "near-stocks domain; session key signs Register (possession proof) and NearWithdraw (exact receiver)",
		"contractAccount": acct,
		"chainId":         chain,
		"sessionKey":      sessionKey,
		"register": map[string]any{
			"subAccountId": hx(subBytes[:]), "userAddress": addr, "sessionKey": sessionKey,
			"expiryTimeStamp": "1790518400000", "nonce": "0", "chainId": chain,
			"digest": hx(rh), "signature": rsig,
		},
		"withdraw": map[string]any{
			"subAccountId": hx(subBytes[:]), "sessionKey": sessionKey, "productId": 4,
			"amount": wd.Amount, "nonce": "3", "receiver": wd.Receiver, "chainId": chain,
			"digest": hx(wh), "signature": wsig,
		},
	})
}
