// NOTE: If you are making any changes to this file then make sure to run the tests in "utils/subaccount.utils_test.go" file

package cutils

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

const MAX_SUBACCOUNT_PER_BROKER = 2

func CreateSubaccountId(brokerId uint, ethAddress string, subaccountNumber int) string {
	return fmt.Sprintf("%v_%v_%v", brokerId, ethAddress, subaccountNumber)
}

func UintTo6Bytes(num uint64) ([6]byte, error) {
	var bytes [6]byte

	if num >= uint64(math.Pow(2, 48)) {
		return bytes, fmt.Errorf("number must be less than %v", uint64(math.Pow(2, 48)))
	}
	for i := 0; i < 6; i++ {
		bytes[5-i] = byte(num >> (i * 8))
	}
	return bytes, nil
}

func EthAddressTo20Bytes(address string) ([20]byte, error) {
	var retBytes [20]byte

	// Check if the address starts with '0x' and remove it if present.
	if len(address) >= 2 && address[:2] == "0x" {
		address = address[2:]
	}

	// Decode the hexadecimal string to bytes.
	bytes, err := hex.DecodeString(address)
	if err != nil {
		return retBytes, err
	}

	// Ensure the length is correct for an Ethereum address.
	if len(bytes) != common.AddressLength {
		return retBytes, fmt.Errorf("invalid address length: expected %d bytes, got %d bytes", common.AddressLength, len(bytes))
	}

	copy(retBytes[:], bytes)

	return retBytes, nil
}

func StakeIdToBytes(id string) ([32]byte, error) {
	var retBytes [32]byte

	if len(id) != 66 || id[:2] != "0x" {
		return retBytes, errors.New("input must be a 32-byte hex string prefixed with '0x'")
	}

	bytes, err := hex.DecodeString(id[2:])
	if err != nil {
		return retBytes, fmt.Errorf("failed to decode hex string: %v", err)
	}

	if len(bytes) != 32 {
		return retBytes, errors.New("decoded bytes length is not 32")
	}

	copy(retBytes[:], bytes)
	return retBytes, nil
}

func SubAccountIdStrToBytes32(id string) ([32]byte, error) {
	var byteArray [32]byte

	byteSlice, _ := hex.DecodeString(id[2:])

	// Check if the string is too long to fit into [32]byte
	if len(byteSlice) > 32 {
		return byteArray, fmt.Errorf("string is too long to convert to [32]byte")
	}

	// Copy byte slice into [32]byte array
	copy(byteArray[:], byteSlice)

	return byteArray, nil
}

func SubaccountIdToBytes32(id string) ([32]byte, error) {
	var retBytes [32]byte

	parts := strings.Split(id, "_")
	brokerId, err1 := strconv.ParseUint(parts[0], 10, 48)
	if err1 != nil {
		return retBytes, fmt.Errorf("broker id must be a number and should be less than 2^48: %v | subaccount_id: %v", err1, id)
	}

	subaccountNum, err2 := strconv.ParseUint(parts[2], 10, 48)
	if err2 != nil {
		return retBytes, fmt.Errorf("subaccount number must be a number and should be less than 2^48: %v", err2)
	}

	ethAddress := parts[1]

	first6Bytes, err3 := UintTo6Bytes(brokerId)
	if err3 != nil {
		return retBytes, err3
	}
	next20Bytes, err4 := EthAddressTo20Bytes(ethAddress)
	if err4 != nil {
		return retBytes, err4
	}
	last6bytes, err5 := UintTo6Bytes(subaccountNum)
	if err5 != nil {
		return retBytes, err5
	}

	bytes := append(append(first6Bytes[:], next20Bytes[:]...), last6bytes[:]...)

	copy(retBytes[:], bytes)
	return retBytes, nil
}

func Bytes32ToSubaccountId(bytes [32]byte) string {
	brokerIdBytes := append([]byte{0, 0}, bytes[0:6]...)
	ethAddressBytes := bytes[6:26]
	subaccountNumBytes := append([]byte{0, 0}, bytes[26:32]...)

	brokerId := binary.BigEndian.Uint64(brokerIdBytes)
	ethAddress := common.BytesToAddress(ethAddressBytes).Hex() // This will return checksum address
	subaccountNum := binary.BigEndian.Uint64(subaccountNumBytes)

	return CreateSubaccountId(uint(brokerId), ethAddress, int(subaccountNum))
}

func HexToSubaccountId(hexString string) (string, error) {
	// Remove the '0x' prefix if present
	if len(hexString) >= 2 && hexString[:2] == "0x" {
		hexString = hexString[2:]
	}

	bytes, err := hex.DecodeString(hexString)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex string: %v", err)
	}

	if len(bytes) != 32 {
		return "", errors.New("decoded bytes length is not 32")
	}

	return Bytes32ToSubaccountId([32]byte(bytes)), nil
}

func Bytes32ToSubaccountHex(bytes [32]byte) string {
	return hexutil.Encode(bytes[:])
}

func ExtractEthereumAddress(subaccountId string) (string, error) {
	// Split the string by "_"
	parts := strings.Split(subaccountId, "_")

	// Ensure that the input string contains at least 2 parts (before and after the address)
	if len(parts) < 2 {
		return "", errors.New("invalid format: cannot extract Ethereum address")
	}

	// Return the second part, which is the Ethereum address
	return parts[1], nil
}

func HexToSubaccountBytes32(hexString string) ([32]byte, error) {
	if len(hexString) >= 2 && hexString[:2] == "0x" {
		hexString = hexString[2:]
	}
	bytes, err := hex.DecodeString(hexString)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to decode hex string: %v", err)
	}

	if len(bytes) != 32 {
		return [32]byte{}, errors.New("decoded bytes length is not 32")
	}

	var byteArray [32]byte
	copy(byteArray[:], bytes)

	return byteArray, nil
}

// Returns the lowercased hex value representation of the string subaccount id
func SubaccountIdToHex(id string) (string, error) {
	bytes, err := SubaccountIdToBytes32(id)
	if err != nil {
		return "", err
	}

	subaccountIdHex := hexutil.Encode(bytes[:])
	return strings.ToLower(subaccountIdHex), nil
}

// If the subaccount id is already in hex format then it will return the same hex value.
// otherwise Returns the hex representation of the string subaccount id.
func HackySubaccountIdToHex(id string) (string, error) {
	if len(id) >= 2 && id[:2] == "0x" {
		return id, nil
	}

	return SubaccountIdToHex(id)
}

// if the subaccount hex is already in string format then it will return the same string value.
// otherwise Returns the string representation of the hex subaccount id.
func HackySubaccountHexToId(hexString string) (string, error) {
	if len(hexString) >= 2 && hexString[:2] == "0x" {
		return HexToSubaccountId(hexString)
	}

	return hexString, nil
}

func ValidateSubaccountId(id string, brokerId uint, ethAddress string) bool {
	parts := strings.Split(id, "_")
	if len(parts) != 3 {
		fmt.Println("Number of parts must be 3 or")
		return false
	}

	// Parse and validate brokerId
	idBrokerId, err := strconv.ParseUint(parts[0], 10, 48)
	if err != nil || uint(idBrokerId) != brokerId {
		fmt.Printf("Broker Id did not match: %v\n", err)
		return false
	}

	// Validate ethAddress
	if parts[1] != ethAddress {
		fmt.Printf("Eth address did not match")
		return false
	}

	// Parse and validate subaccountNumber
	subaccountNumber, err := strconv.ParseUint(parts[2], 10, 48)
	if err != nil || (subaccountNumber < 1 || subaccountNumber > MAX_SUBACCOUNT_PER_BROKER) {
		fmt.Printf("Subaccount number exceeding %v", err)
		return false
	}

	return true
}

func ExtractBrokerIdFromSubaccountHex(subaccountHex string) uint {
	subaccountId, err := HexToSubaccountId(subaccountHex)
	if err != nil {
		return 0
	}
	parts := strings.Split(subaccountId, "_")
	if len(parts) < 3 {
		return 0
	}
	brokerId, _ := strconv.ParseUint(parts[0], 10, 48)
	return uint(brokerId)
}

func ParseSubaccountIDsAsHexsFromString(idsStr, delimiter string) []string {
	idsArray := GetTrimmedSplitValue(idsStr, delimiter)
	subaccountHexs := []string{}

	for _, id := range idsArray {
		hex, err := SubaccountIdToHex(id)
		if err != nil {
			xlog.Errorf("Error converting subaccount ID %s to hex: %v. Skipping\n", id, err)
			continue
		}
		subaccountHexs = append(subaccountHexs, hex)
	}
	return subaccountHexs
}

func ParseSubaccountIDsAsHexsFromEnv(envKey, delimiter string) []string {
	idsStr := os.Getenv(envKey)
	return ParseSubaccountIDsAsHexsFromString(idsStr, delimiter)
}

// IsValidNearAccountId applies the NEAR account-id rules: 2-64 chars of [a-z0-9], with
// single '-', '_' or '.' separators that never start, end or repeat.
func IsValidNearAccountId(accountId string) bool {
	if len(accountId) < 2 || len(accountId) > 64 {
		return false
	}
	prevSeparator := true // treat the start as a separator so a leading one is rejected
	for i := 0; i < len(accountId); i++ {
		c := accountId[i]
		switch {
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'):
			prevSeparator = false
		case c == '-' || c == '_' || c == '.':
			if prevSeparator {
				return false
			}
			prevSeparator = true
		default:
			return false
		}
	}
	return !prevSeparator
}

// NearAccountToAddr20 maps a NEAR account to the 20-byte address slot of the subaccount layout
// (Development.md §6.1): keccak256("near:" ‖ account_id)[12..32], returned as a checksummed hex
// address so it drops into CreateSubaccountId / SubaccountIdToBytes32 unchanged.
func NearAccountToAddr20(accountId string) (string, error) {
	if !IsValidNearAccountId(accountId) {
		return "", fmt.Errorf("invalid NEAR account id: %q", accountId)
	}
	hash := crypto.Keccak256([]byte("near:" + accountId))
	return common.BytesToAddress(hash[12:]).Hex(), nil
}
