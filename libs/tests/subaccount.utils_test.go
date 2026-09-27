package tests

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"math"
	"testing"
)

func TestSubaccountIdConversion(t *testing.T) {
	for i := 1; i < int(math.Pow(2, 12)); i++ {
		subaccountId := fmt.Sprintf("%v_0x22a06BCbe144414399117ae98Be847bdC79F0dC3_%v", i, i)
		bytes, err := cutils.SubaccountIdToBytes32(subaccountId)
		if err != nil {
			t.Fatal(err)
		}
		decodeSubaccountId := cutils.Bytes32ToSubaccountId(bytes)
		if subaccountId != decodeSubaccountId {
			t.Fatalf("Unexpected subaccountId: %v, decodedSubaccountId: %v", subaccountId, decodeSubaccountId)
		}
	}
}

func TestSubaccountIdToHex(t *testing.T) {
	subaccountId := "1_0x899e72b11fed23e91a70789a55e1f56bdb8d2a8e_1"
	subaccountIdHex, err := cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("subaccountIdHex: %v\n", subaccountIdHex)
	if subaccountIdHex != "0x000000000001899e72b11fed23e91a70789a55e1f56bdb8d2a8e000000000001" {
		t.Fatalf("Unexpected subaccountId: %v, subaccountIdHex: %v", subaccountId, subaccountIdHex)
	}

	// Test 2
	subaccountId = "1_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_1"
	subaccountIdHex, err = cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("subaccountIdHex: %v\n", subaccountIdHex)
	if subaccountIdHex != "0x000000000001a7de990d10a7d8b53476525b1212bacf645cb7b0000000000001" {
		t.Fatalf("Unexpected subaccountId: %v, subaccountIdHex: %v", subaccountId, subaccountIdHex)
	}

	// Test 3
	subaccountId = "1_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_1"
	subaccountIdHex, err = cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("subaccountIdHex: %v\n", subaccountIdHex)
	if subaccountIdHex != "0x000000000001a7de990d10a7d8b53476525b1212bacf645cb7b0000000000001" {
		t.Fatalf("Unexpected subaccountId: %v, subaccountIdHex: %v", subaccountId, subaccountIdHex)
	}
}

func TestHexToSubaccountId(t *testing.T) {
	// Test 1 : Hex is with checksum eth address
	hex := "000000000001818484227ABF04550c6c242B6119B7c94d2E72b3000000000001"
	subaccountId, err := cutils.HexToSubaccountId(hex)
	if err != nil {
		t.Fatal(err)
	}

	if subaccountId != "1_0x818484227ABF04550c6c242B6119B7c94d2E72b3_1" {
		t.Fatalf("Unexpected subaccountId: %v", subaccountId)
	}

	// Test 2 : Hex is lowercased
	hex = "000000000001a7de990d10a7d8b53476525b1212bacf645cb7b0000000000001"
	subaccountId, err = cutils.HexToSubaccountId(hex)
	if err != nil {
		t.Fatal(err)
	}

	if subaccountId != "1_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_1" {
		t.Fatalf("Unexpected subaccountId: %v", subaccountId)
	}
}

func TestHexToSubaccountBytes32(t *testing.T) {
	// Test 1 : Hex is with checksum eth address
	hex := "0x000000000001076c746c2fc735c478e3c9f456e00d25d63527e2000000000001"
	bytes, err := cutils.HexToSubaccountBytes32(hex)
	if err != nil {
		t.Fatal(err)
	}
	hexString := cutils.Bytes32ToSubaccountHex(bytes)
	if hexString != hex {
		t.Fatalf("Unexpected hexString: %v", hexString)
	}
}

func TestExtractBrokerIdFromSubaccountHex(t *testing.T) {
	// Test 1: Valid subaccount hex with brokerId = 1
	subaccountId := "1_0x899e72b11fed23e91a70789a55e1f56bdb8d2a8e_1"
	subaccountIdHex, err := cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		t.Fatalf("Failed to convert subaccountId to hex: %v", err)
	}
	brokerId := cutils.ExtractBrokerIdFromSubaccountHex(subaccountIdHex)
	if brokerId != 1 {
		t.Fatalf("Expected brokerId 1, got %v", brokerId)
	}

	// Test 2: Valid subaccount hex with brokerId = 12345
	subaccountId = "2_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_1"
	subaccountIdHex, err = cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		t.Fatalf("Failed to convert subaccountId to hex: %v", err)
	}
	brokerId = cutils.ExtractBrokerIdFromSubaccountHex(subaccountIdHex)
	if brokerId != 2 {
		t.Fatalf("Expected brokerId 12345, got %v", brokerId)
	}

	// Test 3: Invalid hex string (not 32 bytes)
	invalidHex := "0xdeadbeef"
	brokerId = cutils.ExtractBrokerIdFromSubaccountHex(invalidHex)
	if brokerId != 0 {
		t.Fatalf("Expected brokerId 0 for invalid hex, got %v", brokerId)
	}

	// Test 4: Hex string with missing brokerId part after decoding
	// This is a malformed hex that decodes to a string with less than 3 parts
	malformedHex := "0x0000000000000000000000000000000000000000000000000000000000000000"
	brokerId = cutils.ExtractBrokerIdFromSubaccountHex(malformedHex)
	if brokerId != 0 {
		t.Fatalf("Expected brokerId 0 for malformed hex, got %v", brokerId)
	}
}

func TestStringHexToStringId(t *testing.T) {
	subaccountHexs := "0x00000000000165ddd96914436e084f1cc332c7efb100ecb980b3000000000001,0x0000000000019c93f5c8afe5e5b2642ec76e0edfec34dadcfe2d000000000001,0x000000000001439d907b94437bad645697a903f8610114563571000000000001,0x000000000001287de2ac7a77bd0dd13886e570e1a089e21ed402000000000001,0x00000000000119cbc6df7f74d6e75cead93a25dece671ce985f8000000000001,0x000000000001582628702b25af512a88782f60c1b6bcd63c8666000000000001,0x0000000000011c4039855b68e3a3cf27d9695268f92b834d7739000000000001,0x000000000001e84e8f2da14c4f6c1460d6973bed37776fce8988000000000001,0x000000000001876af73bbf695c8b7d079b82e3ff84e0bd49404b000000000001,0x000000000001a5d6823b8f57477bd7d12662b50096f508f1c926000000000001,0x000000000001f1d39cf8fd9086d943490063e441d9fa2afea6d9000000000001,0x0000000000028221a68d129f70877750029d9d45a58e395111ec000000000001,0x000000000002a548dc4ce939ebc62e55f154293357fd50a32f0c000000000001,0x0000000000028521953d26f17a3ae0b2b99739757e37d68abd2a000000000001,0x00000000000148e7db25cb247bf02b25402108e4499a6636eaa9000000000001,0x000000000001ebe09b0c96b109039745b3c9bbd56e194d9b5783000000000001,0x0000000000015defe4f60965528030ee48bba305ed017ab84998000000000001,0x000000000001bcc830633a7af4ac054491333b62e611ca55b94d000000000001,0x000000000001c73d55ec17c806709def8fbb3bb3bcb299383d56000000000001,0x000000000001b3023baa698dd410a8afcfae5f104005ed7d71bf000000000001,0x000000000001a3280c1260d7be3cd9bf93e3ff7087d09fd1821c000000000001"
	expectedIds := "1_0x65ddd96914436E084F1Cc332c7eFb100ecb980B3_1,1_0x9C93f5C8aFe5e5b2642eC76E0EDFEc34DAdCFE2D_1,1_0x439d907B94437BAD645697a903f8610114563571_1,1_0x287dE2AC7a77Bd0DD13886e570e1A089e21Ed402_1,1_0x19CbC6Df7F74d6e75cEad93a25deCE671Ce985f8_1,1_0x582628702B25AF512A88782F60C1b6BcD63C8666_1,1_0x1c4039855b68E3A3Cf27d9695268f92B834D7739_1,1_0xe84e8f2dA14C4F6C1460D6973bED37776FCE8988_1,1_0x876aF73bbF695c8B7d079B82e3fF84e0BD49404B_1,1_0xa5D6823b8F57477Bd7D12662B50096F508f1c926_1,1_0xf1d39Cf8FD9086D943490063e441D9FA2afea6D9_1,2_0x8221a68d129f70877750029D9D45a58E395111Ec_1,2_0xA548dC4ce939eBC62e55F154293357fD50A32f0C_1,2_0x8521953D26F17A3aE0B2B99739757e37d68abd2a_1,1_0x48e7DB25cB247Bf02B25402108e4499A6636EAa9_1,1_0xEBe09B0C96B109039745B3C9BBD56E194d9b5783_1,1_0x5defe4f60965528030EE48bba305Ed017AB84998_1,1_0xbCc830633A7AF4aC054491333B62E611ca55B94D_1,1_0xc73D55eC17c806709def8FbB3bB3BCb299383d56_1,1_0xb3023BAa698DD410a8AfCFAE5F104005eD7d71Bf_1,1_0xa3280C1260D7be3Cd9Bf93e3fF7087D09Fd1821c_1"
	subaccountIds := ""
	hexArray := cutils.GetTrimmedSplitValue(subaccountHexs, ",")
	for _, subaccountHex := range hexArray {
		subaccountId, err := cutils.HexToSubaccountId(subaccountHex)
		if err != nil {
			t.Fatal(err)
		}
		subaccountIds += subaccountId + ","
	}
	subaccountIds = subaccountIds[:len(subaccountIds)-1] // Remove trailing comma
	if subaccountIds != expectedIds {
		t.Fatalf("Unexpected subaccountIds: %v", subaccountIds)
	}
}

func TestParseSubaccountIDsHexFromString(t *testing.T) {
	subaccountIds := "1_0x65ddd96914436E084F1Cc332c7eFb100ecb980B3_1,1_0x9C93f5C8aFe5e5b2642eC76E0EDFEc34DAdCFE2D_1,1_0x439d907B94437BAD645697a903f8610114563571_1,1_0x287dE2AC7a77Bd0DD13886e570e1A089e21Ed402_1,1_0x19CbC6Df7F74d6e75cEad93a25deCE671Ce985f8_1,1_0x582628702B25AF512A88782F60C1b6BcD63C8666_1,1_0x1c4039855b68E3A3Cf27d9695268f92B834D7739_1,1_0xe84e8f2dA14C4F6C1460D6973bED37776FCE8988_1,1_0x876aF73bbF695c8B7d079B82e3fF84e0BD49404B_1,1_0xa5D6823b8F57477Bd7D12662B50096F508f1c926_1,1_0xf1d39Cf8FD9086D943490063e441D9FA2afea6D9_1,2_0x8221a68d129f70877750029D9D45a58E395111Ec_1,2_0xA548dC4ce939eBC62e55F154293357fD50A32f0C_1,2_0x8521953D26F17A3aE0B2B99739757e37d68abd2a_1,1_0x48e7DB25cB247Bf02B25402108e4499A6636EAa9_1,1_0xEBe09B0C96B109039745B3C9BBD56E194d9b5783_1,1_0x5defe4f60965528030EE48bba305Ed017AB84998_1,1_0xbCc830633A7AF4aC054491333B62E611ca55B94D_1,1_0xc73D55eC17c806709def8FbB3bB3BCb299383d56_1,1_0xb3023BAa698DD410a8AfCFAE5F104005eD7d71Bf_1,1_0xa3280C1260D7be3Cd9Bf93e3fF7087D09Fd1821c_1"
	expectedHexs := "0x00000000000165ddd96914436e084f1cc332c7efb100ecb980b3000000000001,0x0000000000019c93f5c8afe5e5b2642ec76e0edfec34dadcfe2d000000000001,0x000000000001439d907b94437bad645697a903f8610114563571000000000001,0x000000000001287de2ac7a77bd0dd13886e570e1a089e21ed402000000000001,0x00000000000119cbc6df7f74d6e75cead93a25dece671ce985f8000000000001,0x000000000001582628702b25af512a88782f60c1b6bcd63c8666000000000001,0x0000000000011c4039855b68e3a3cf27d9695268f92b834d7739000000000001,0x000000000001e84e8f2da14c4f6c1460d6973bed37776fce8988000000000001,0x000000000001876af73bbf695c8b7d079b82e3ff84e0bd49404b000000000001,0x000000000001a5d6823b8f57477bd7d12662b50096f508f1c926000000000001,0x000000000001f1d39cf8fd9086d943490063e441d9fa2afea6d9000000000001,0x0000000000028221a68d129f70877750029d9d45a58e395111ec000000000001,0x000000000002a548dc4ce939ebc62e55f154293357fd50a32f0c000000000001,0x0000000000028521953d26f17a3ae0b2b99739757e37d68abd2a000000000001,0x00000000000148e7db25cb247bf02b25402108e4499a6636eaa9000000000001,0x000000000001ebe09b0c96b109039745b3c9bbd56e194d9b5783000000000001,0x0000000000015defe4f60965528030ee48bba305ed017ab84998000000000001,0x000000000001bcc830633a7af4ac054491333b62e611ca55b94d000000000001,0x000000000001c73d55ec17c806709def8fbb3bb3bcb299383d56000000000001,0x000000000001b3023baa698dd410a8afcfae5f104005ed7d71bf000000000001,0x000000000001a3280c1260d7be3cd9bf93e3ff7087d09fd1821c000000000001"

	result := cutils.ParseSubaccountIDsAsHexsFromString(subaccountIds, ",")
	subaccountHexs := ""
	for _, hex := range result {
		subaccountHexs += hex + ","
	}
	if len(subaccountHexs) > 0 {
		subaccountHexs = subaccountHexs[:len(subaccountHexs)-1] // Remove trailing comma
	}

	if subaccountHexs != expectedHexs {
		t.Fatalf("Unexpected subaccountHexs: %v", subaccountHexs)
	}
}
