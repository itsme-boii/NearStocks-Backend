package cutils

import "testing"

func TestIsValidNearAccountId(t *testing.T) {
	valid := []string{"ab", "alice.near", "near-stocks.near", "a1_b-c.sub.near", "0x5a4a3f0fcd06cd2d3e5f7d18ea4a3c6b2de0ae7c",
		"98793cd91a3f870fb126f66285808c7e094afcfc4eda8a970f6648cdf0dbd6de"}
	invalid := []string{"", "a", "Alice.near", ".near", "alice..near", "alice.", "a b", "alice.near-", "-a", "a_-b",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} // 65 chars
	for _, a := range valid {
		if !IsValidNearAccountId(a) {
			t.Errorf("%q should be valid", a)
		}
	}
	for _, a := range invalid {
		if IsValidNearAccountId(a) {
			t.Errorf("%q should be invalid", a)
		}
	}
}

func TestNearAccountToAddr20RoundTrip(t *testing.T) {
	addr, err := NearAccountToAddr20("alice.near")
	if err != nil {
		t.Fatal(err)
	}
	if addr != "0xfB734EC1D441bFAee35EdDb6DDdB0774b8F0ec67" { // vectors/addr20.json
		t.Fatalf("addr20 = %s", addr)
	}
	id := CreateSubaccountId(1, addr, 0)
	b, err := SubaccountIdToBytes32(id)
	if err != nil {
		t.Fatal(err)
	}
	if Bytes32ToSubaccountId(b) != id {
		t.Fatalf("round trip %s != %s", Bytes32ToSubaccountId(b), id)
	}
}
