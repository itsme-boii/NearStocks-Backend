package tests

import (
	"errors"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/logxTokenUtils"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTokenVesting(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TokenVesting Suite")
}

var _ = Describe("Token Vesting", func() {
	var tokenUtils *logxTokenUtils.LogxTokenUtils
	var vesting db.TokenVestingTable

	// Setup before each test
	BeforeEach(func() {
		// Setup vesting data
		vesting = db.TokenVestingTable{
			BaseTable: db.BaseTable{
				CreatedAt: time.Date(2024, 9, 23, 0, 0, 0, 0, time.UTC), // Fixed time
				UpdatedAt: time.Date(2024, 9, 23, 0, 0, 0, 0, time.UTC), // Fixed time
			},
			UserAddress:          "0x1234",
			TotalAmt:             ctypes.NewBigInt(cutils.Mulx18(big.NewInt(1000))),
			ClaimedAmt:           ctypes.NewBigInt(big.NewInt(0)),
			AmtUnlockDay1:        ctypes.NewBigInt(cutils.Mulx18(big.NewInt(200))),
			UnlockPercentageX100: ctypes.NewBigInt(big.NewInt(333)), /// this is 3.33 %
			Frequency:            30,                                /// means monthly unlock
			VestingType:          "KOL",
		}
	})

	DescribeTable("Testing different vesting scenarios",
		func(currentTime time.Time, expectedClaimable ctypes.BigInt, expectedNextUnlock ctypes.BigInt, expectedTimeRemaining int64) {
			// Call the CalculateUnclaimedVestedAmount function with the provided current time
			unlockedAmount, timeRemaining, nextUnlockAmount := tokenUtils.CalculateUnclaimedVestedAmount(vesting, currentTime)

			// Assert the unlocked amount matches the expected value
			Expect(unlockedAmount.Cmp(expectedClaimable)).To(Equal(0), "Claimable amount mismatch")

			// Assert the next unlock amount matches the expected value
			Expect(nextUnlockAmount.Cmp(expectedNextUnlock)).To(Equal(0), "Next unlock amount mismatch")

			// Assert the time remaining matches the expected value
			Expect(int64(timeRemaining)).To(Equal(expectedTimeRemaining), "Time remaining mismatch")
		},

		// // Test case 1: Initial unlock scenario on 2024-09-24
		Entry("First unlock on 2024-09-24 12:00 UTC",
			time.Date(2024, 9, 23, 12, 0, 0, 0, time.UTC),           // Current time for the test
			ctypes.NewBigInt(cutils.Mulx18(big.NewInt(200))),        // Expected claimable amount --- day 1 unlock amount
			ctypes.NewBigInt(cutils.MulxCust(big.NewInt(2664), 16)), // Expected next unlock amount --- next month unlock amount
			int64(720), // Expected time remaining in minutes --- 29 days 12 hrs = 42480 minutes
		),

		// // Test case 2: Second unlock scenario on 2024-10-24
		Entry("Second unlock on 2024-10-30 12:00 UTC",
			time.Date(2024, 10, 29, 12, 0, 0, 0, time.UTC),           // Current time for the test
			ctypes.NewBigInt(cutils.MulxCust(big.NewInt(22664), 16)), // Expected claimable amount---- day1 + last month amount
			ctypes.NewBigInt(cutils.MulxCust(big.NewInt(2664), 16)),  // Expected next unlock amount ---- next month unlock amount
			int64(36720), // Expected time remaining in minutes  --- 25 days 12hrs = 35280
		),
	)

	// Add daily vesting tests
	Context("Testing daily vesting scenarios", func() {
		BeforeEach(func() {
			// Setup vesting data for daily unlocks
			vesting = db.TokenVestingTable{
				BaseTable: db.BaseTable{
					CreatedAt: time.Date(2024, 9, 23, 0, 0, 0, 0, time.UTC), // Fixed time
					UpdatedAt: time.Date(2024, 9, 23, 0, 0, 0, 0, time.UTC), // Fixed time
				},
				UserAddress:          "0x1234",
				TotalAmt:             ctypes.NewBigInt(cutils.Mulx18(big.NewInt(1000))),
				ClaimedAmt:           ctypes.NewBigInt(big.NewInt(0)),
				AmtUnlockDay1:        ctypes.NewBigInt(cutils.Mulx18(big.NewInt(200))),
				UnlockPercentageX100: ctypes.NewBigInt(big.NewInt(333)), // 3.33 %
				Frequency:            1,                                 // Daily unlock
				VestingType:          "KOL",
			}
		})

		DescribeTable("Testing daily vesting scenarios",
			func(currentTime time.Time, expectedClaimable ctypes.BigInt, expectedNextUnlock ctypes.BigInt, expectedTimeRemaining int64) {
				// Call the CalculateUnclaimedVestedAmount function with the provided current time
				unlockedAmount, timeRemaining, nextUnlockAmount := tokenUtils.CalculateUnclaimedVestedAmount(vesting, currentTime)
				// Assert the unlocked amount matches the expected value
				Expect(unlockedAmount.Cmp(expectedClaimable)).To(Equal(0), "Claimable amount mismatch")

				// Assert the next unlock amount matches the expected value
				Expect(nextUnlockAmount.Cmp(expectedNextUnlock)).To(Equal(0), "Next unlock amount mismatch")
				// Assert the time remaining matches the expected value
				Expect(int64(timeRemaining)).To(Equal(expectedTimeRemaining), "Time remaining mismatch")
			},

			// Test case 1: Initial unlock scenario on 2024-09-24
			Entry("First unlock on 2024-09-24 12:00 UTC for daily vesting",
				time.Date(2024, 9, 23, 12, 0, 0, 0, time.UTC),           // Current time for the test
				ctypes.NewBigInt(cutils.Mulx18(big.NewInt(200))),        // Expected claimable amount --- day 1 unlock amount
				ctypes.NewBigInt(cutils.MulxCust(big.NewInt(2664), 16)), // Expected next unlock amount --- next day unlock amount
				int64(720), // Expected time remaining in minutes --- 12 hours = 720 minutes
			),

			// Test case 2: Claim scenario on 2024-09-25
			Entry("Claim and check on 2024-09-25 00:01 UTC for daily vesting",
				time.Date(2024, 9, 24, 0, 1, 0, 0, time.UTC),             // Current time for the test
				ctypes.NewBigInt(cutils.MulxCust(big.NewInt(22664), 16)), // Expected claimable amount---- day1 + last few days' amount
				ctypes.NewBigInt(cutils.MulxCust(big.NewInt(2664), 16)),  // Expected next unlock amount ---- next day unlock amount
				int64(1439), // Expected time remaining in minutes --- 12 hours = 720 minutes
			),
		)
	})

	DescribeTable("Testing Consecutive claiming by user in daily vesting",
		func(currentTime time.Time, claimedAmount ctypes.BigInt, expectedClaimable ctypes.BigInt, expectedNextUnlock ctypes.BigInt, expectedTimeRemaining int64) {

			// Setup vesting data for each entry
			vesting = db.TokenVestingTable{
				BaseTable: db.BaseTable{
					CreatedAt: time.Date(2024, 9, 23, 0, 0, 0, 0, time.UTC), // Fixed time
					UpdatedAt: time.Date(2024, 9, 23, 0, 0, 0, 0, time.UTC), // Fixed time
				},
				UserAddress:          "0x1234",
				TotalAmt:             ctypes.NewBigInt(cutils.Mulx18(big.NewInt(1000))),
				ClaimedAmt:           claimedAmount, // Use claimedAmount for each test case
				AmtUnlockDay1:        ctypes.NewBigInt(cutils.Mulx18(big.NewInt(200))),
				UnlockPercentageX100: ctypes.NewBigInt(big.NewInt(333)), //
				Frequency:            1,                                 // Daily unlock
				VestingType:          "daily",
			}

			// Call the CalculateUnclaimedVestedAmount function with the provided current time
			unlockedAmount, timeRemaining, nextUnlockAmount := tokenUtils.CalculateUnclaimedVestedAmount(vesting, currentTime)
			// Assert the unlocked amount matches the expected value
			Expect(unlockedAmount.Cmp(expectedClaimable)).To(Equal(0), "Claimable amount mismatch")

			// Assert the next unlock amount matches the expected value
			Expect(nextUnlockAmount.Cmp(expectedNextUnlock)).To(Equal(0), "Next unlock amount mismatch")

			// Assert the time remaining matches the expected value
			Expect(int64(timeRemaining)).To(Equal(expectedTimeRemaining), "Time remaining mismatch")

		},

		// Chain Test Case 1: User comes on 2024-09-24 and claims the initial unlock amount
		Entry("User claims on 2024-09-24 12:00 UTC",
			time.Date(2024, 9, 23, 12, 0, 0, 0, time.UTC),           // Current time for the test
			ctypes.NewBigInt(big.NewInt(0)),                         // Initial claimed amount is 0
			ctypes.NewBigInt(cutils.Mulx18(big.NewInt(200))),        // Expected claimable amount --- day 1 unlock amount
			ctypes.NewBigInt(cutils.MulxCust(big.NewInt(2664), 16)), // Expected next unlock amount --- next day unlock amount
			int64(720), // Expected time remaining in minutes --- 12 hours = 720 minutes
		),

		// Chain Test Case 2: User comes on 2024-09-26 and claims again, after already claiming on 2024-09-24
		Entry("User claims again on 2024-09-26 12:00 UTC",
			time.Date(2024, 9, 25, 12, 0, 0, 0, time.UTC),           // Current time for the test
			ctypes.NewBigInt(cutils.Mulx18(big.NewInt(200))),        // Previously claimed 200 tokens
			ctypes.NewBigInt(cutils.MulxCust(big.NewInt(5328), 16)), // Expected claimable amount --- 2 more daily unlocks
			ctypes.NewBigInt(cutils.MulxCust(big.NewInt(2664), 16)), // Expected next unlock amount --- next day unlock amount
			int64(720), // Expected time remaining in minutes --- 12 hours = 720 minutes
		),

		// Chain Test Case 3: User comes on 2024-10-24 and claims after one month
		Entry("User claims after 1 month on 2024-10-24 12:00 UTC",
			time.Date(2024, 10, 23, 12, 0, 0, 0, time.UTC),          // Current time for the test
			ctypes.NewBigInt(cutils.Mulx18(big.NewInt(253))),        // Previously claimed 253 tokens (200+53)
			ctypes.NewBigInt(cutils.MulxCust(big.NewInt(7462), 17)), // Expected claimable amount --- unlocked over 1 month that complete got unlocked as 24 days vesting
			ctypes.NewBigInt(cutils.MulxCust(big.NewInt(2664), 16)), // Expected next unlock amount --- next daily unlock amount - 0
			int64(720), // Expected time remaining in minutes ---0 // this handling done in another function
		),
	)

	Describe("User having both Vesting token and Trading Token", func() {
		var tokenUtils *logxTokenUtils.LogxTokenUtils
		var tokenUserData db.LogxTokenUserTable
		var vesting db.TokenVestingTable
		var vestingData []db.TokenVestingTable

		BeforeEach(func() {
			// Initialize user data
			tokenUserData = db.LogxTokenUserTable{
				UserAddress: "0x1234",
				TotalAmt:    ctypes.NewBigInt(big.NewInt(1000)),
				ClaimedAmt:  ctypes.NewBigInt(big.NewInt(0)),
				HasVesting:  true,
				IsBlocked:   false,
			}

			// Initialize vesting data
			vesting = db.TokenVestingTable{
				UserAddress:          "0x1234",
				TotalAmt:             ctypes.NewBigInt(big.NewInt(1000)),
				ClaimedAmt:           ctypes.NewBigInt(big.NewInt(0)),
				AmtUnlockDay1:        ctypes.NewBigInt(big.NewInt(200)),
				UnlockPercentageX100: ctypes.NewBigInt(big.NewInt(333)), // 3.33%
				Frequency:            1,                                 // Daily unlock
				VestingType:          "KOL",
			}

			vestingData = []db.TokenVestingTable{vesting}
		})

		DescribeTable("Chain Vesting Testing",
			func(timeStep time.Time, expectedClaimable ctypes.BigInt, expectedNextUnlock ctypes.BigInt, expectedTimeRemaining int64) {
				// Call the ClaimableAmountifVesting function and get the results
				totalClaimableAmount, _, minTimeRemaining, nextUnlockAmount, _, _ := tokenUtils.ClaimableAmountifVesting(&tokenUserData, vestingData, timeStep)
				// Assert the claimed amount matches the expected value
				Expect(totalClaimableAmount.Cmp(expectedClaimable)).To(Equal(0), "Claimable amount mismatch")

				// Assert the next unlock amount matches the expected value
				Expect(nextUnlockAmount.Cmp(expectedNextUnlock)).To(Equal(0), "Next unlock amount mismatch")

				// Assert the time remaining matches the expected value
				Expect(minTimeRemaining).To(Equal(expectedTimeRemaining), "Time remaining mismatch")

			},

			// Test case 1: Initial unlock scenario on 2024-09-24, where user claims the day 1 amount
			Entry("Initial unlock on 2024-09-24",
				time.Date(2024, 9, 23, 12, 0, 0, 0, time.UTC), // Current time for the test
				ctypes.NewBigInt(big.NewInt(1200)),            // Expected claimable amount (Day 1 unlock Vesting) + 1000 Normal Trading
				ctypes.NewBigInt(big.NewInt(26)),              // Expected next unlock amount
				int64(720),                                    // Expected time remaining (12 hours)
			),

			// Test case 2: User comes 2 days later and claims for first time
			Entry("Second claim after 2 days on 2024-09-26",
				time.Date(2024, 9, 25, 12, 0, 0, 0, time.UTC), // Current time for the test
				ctypes.NewBigInt(big.NewInt(1253)),            // Expected claimable amount (Next 2 days unlock: 2 * 22) + day1 unlock + Normal trading
				ctypes.NewBigInt(big.NewInt(26)),              // Expected next unlock amount
				int64(720),                                    // Expected time remaining (12 hours)
			),

			// // Test case 3: User comes after a month for first Time
			Entry("Claim after 1 month on 2024-10-24",
				time.Date(2024, 11, 23, 12, 0, 0, 0, time.UTC), // Current time for the test
				ctypes.NewBigInt(big.NewInt(1999)),             // Expected claimable amount (Total unlock minus claimed)
				ctypes.NewBigInt(big.NewInt(0)),                // Expected next unlock amount (fully unlocked)
				int64(0),                                       // No time remaining (all tokens unlocked)
			),
		)
	})

	Describe("Token Vesting with Multiple Vesting Types", func() {
		var tokenUtils *logxTokenUtils.LogxTokenUtils
		var tokenUserData db.LogxTokenUserTable
		var vestingData []db.TokenVestingTable
		var vesting1, vesting2 db.TokenVestingTable

		// Setup before each test
		BeforeEach(func() {
			// Initialize user data
			tokenUserData = db.LogxTokenUserTable{
				UserAddress: "0x1234",
				TotalAmt:    ctypes.NewBigInt(big.NewInt(1000)),
				ClaimedAmt:  ctypes.NewBigInt(big.NewInt(0)),
				HasVesting:  true,
				IsBlocked:   false,
			}

			// Initialize first vesting data
			vesting1 = db.TokenVestingTable{
				UserAddress:          "0x1234",
				TotalAmt:             ctypes.NewBigInt(big.NewInt(1000)),
				ClaimedAmt:           ctypes.NewBigInt(big.NewInt(0)),
				AmtUnlockDay1:        ctypes.NewBigInt(big.NewInt(200)),
				UnlockPercentageX100: ctypes.NewBigInt(big.NewInt(333)), // 3.33% daily unlock
				Frequency:            1,                                 // Daily unlock
				VestingType:          "KOL",
			}

			// Initialize second vesting data
			vesting2 = db.TokenVestingTable{
				UserAddress:          "0x1234",
				TotalAmt:             ctypes.NewBigInt(big.NewInt(1000)),
				ClaimedAmt:           ctypes.NewBigInt(big.NewInt(0)),
				AmtUnlockDay1:        ctypes.NewBigInt(big.NewInt(100)),
				UnlockPercentageX100: ctypes.NewBigInt(big.NewInt(666)), // 6.66% daily unlock
				Frequency:            1,                                 // Daily unlock
				VestingType:          "DEAL",
			}

			// Combine both vesting entries for the user
			vestingData = []db.TokenVestingTable{vesting1, vesting2}
		})

		DescribeTable("Testing different vesting types for the same user",
			func(currentTime time.Time, expectedClaimable ctypes.BigInt, expectedNextUnlock ctypes.BigInt, expectedTimeRemaining int64) {
				// Call the ClaimableAmountifVesting function with the provided current time
				totalClaimableAmount, _, timeRemaining, nextUnlockAmount, _, err := tokenUtils.ClaimableAmountifVesting(&tokenUserData, vestingData, currentTime)
				// Ensure no error occurred
				Expect(err).To(BeNil(), "No error should occur")

				// Assert the total claimable amount matches the expected value
				Expect(totalClaimableAmount.Cmp(expectedClaimable)).To(Equal(0), "Claimable amount mismatch")

				// Assert the next unlock amount matches the expected value
				Expect(nextUnlockAmount.Cmp(expectedNextUnlock)).To(Equal(0), "Next unlock amount mismatch")

				// Assert the time remaining matches the expected value
				Expect(timeRemaining).To(Equal(expectedTimeRemaining), "Time remaining mismatch")
			},

			// Test case: User comes on day 1 (2024-09-24 12:00 UTC)
			Entry("User claims on 2024-09-24 12:00 UTC with multiple vesting types",
				time.Date(2024, 9, 23, 12, 0, 0, 0, time.UTC), // Current time
				ctypes.NewBigInt(big.NewInt(1300)),            // Expected claimable amount (200 from vesting1 + 100 from vesting2) + 100 Normal Trading
				ctypes.NewBigInt(big.NewInt(85)),              // Expected next unlock amount (next day unlock: 26 from vesting1 + 6.66% of 900 from vesting2)
				int64(720),                                    // Expected time remaining in minutes (12 hours = 720 minutes)
			),

			// Test case: User comes after a month (2024-10-24 12:00 UTC)
			Entry("User claims after 1 month on 2024-10-24 12:00 UTC with multiple vesting types",
				time.Date(2024, 10, 23, 12, 0, 0, 0, time.UTC), // Current time
				ctypes.NewBigInt(big.NewInt(2998)),             // Expected claimable amount (1000 from vesting1 + 1000 from vesting2) + 1000 Normal
				ctypes.NewBigInt(big.NewInt(0)),                // Expected next unlock amount (next day unlock: 0 from vesting1 + 0 from vesting2)
				int64(0),                                       // Expected time remaining in minutes -- 0 all calimed
			),
		)
	})

	/// There is some issue coming while patching need to fix

	// Describe("GetTokenUserDataByUserAddress when user don't have vesting", func() {
	// 	var tokenUtils *logxTokenUtils.LogxTokenUtils
	// 	var patches *gomonkey.Patches
	// 	var userAddress string

	// 	BeforeEach(func() {
	// 		// Initialize the LogxTokenUtils
	// 		tokenUtils = &logxTokenUtils.LogxTokenUtils{
	// 			TokenDB: &db.LogxTokenDB{}, // Make sure TokenDB is initialized here
	// 		}

	// 		// User address for the tests
	// 		userAddress = "0x1234"
	// 	})

	// 	AfterEach(func() {
	// 		// Reset patches after each test
	// 		if patches != nil {
	// 			patches.Reset()
	// 		}
	// 	})

	// 	Context("When HasVesting is false", func() {
	// 		BeforeEach(func() {
	// 			// Mock the database call to return a user without vesting
	// 			patches = gomonkey.ApplyFunc(tokenUtils.TokenDB.GetLogxTokenUserByUserAddress, func(userAddress string) (*db.LogxTokenUserTable, error) {
	// 				return &db.LogxTokenUserTable{
	// 					UserAddress: "0x1234",
	// 					TotalAmt:    ctypes.NewBigInt(big.NewInt(1000)),
	// 					ClaimedAmt:  ctypes.NewBigInt(big.NewInt(500)),
	// 					HasVesting:  false,
	// 					IsBlocked:   false,
	// 				}, nil
	// 			})

	// 			// Mock the IP blocking check
	// 			patches.ApplyMethod(reflect.TypeOf(tokenUtils), "IsIPBlocked", func(_ *db.LogxTokenDB, userAddress string) (bool, error) {
	// 				return false, nil
	// 			})
	// 		})

	// 		It("should return the correct total and claimable amounts when user is not blocked", func() {
	// 			// Call the function
	// 			response, err := tokenUtils.GetTokenUserDataByUserAddress(userAddress)

	// 			// Expect no error
	// 			Expect(err).To(BeNil())

	// 			// Verify the returned values
	// 			Expect(response.TotalAmount.Cmp(ctypes.NewBigInt(big.NewInt(1000)))).To(Equal(0))
	// 			Expect(response.ClaimableAmount.Cmp(ctypes.NewBigInt(big.NewInt(500)))).To(Equal(0))
	// 			Expect(response.HasVesting).To(BeFalse())
	// 			Expect(response.IsBlocked).To(BeFalse())
	// 		})

	// 		It("should return blocked response when IP is blocked", func() {
	// 			// Mock the IP blocking check to return true (blocked)
	// 			patches.ApplyMethod(reflect.TypeOf(tokenUtils), "IsIPBlocked", func(_ *logxTokenUtils.LogxTokenUtils, userAddress string) (bool, error) {
	// 				return true, nil
	// 			})

	// 			// Call the function
	// 			response, err := tokenUtils.GetTokenUserDataByUserAddress(userAddress)

	// 			// Expect no error
	// 			Expect(err).To(BeNil())

	// 			// Verify the blocked response
	// 			Expect(response.TotalAmount.Cmp(ctypes.NewBigInt(big.NewInt(0)))).To(Equal(0))
	// 			Expect(response.ClaimableAmount.Cmp(ctypes.NewBigInt(big.NewInt(0)))).To(Equal(0))
	// 			Expect(response.HasVesting).To(BeFalse())
	// 			Expect(response.IsBlocked).To(BeTrue())
	// 		})
	// 	})
	// })

	Describe("UpdateClaimedAmtInVesting", func() {
		var tokenUtils *logxTokenUtils.LogxTokenUtils
		var tokenUserData db.LogxTokenUserTable
		var vestingData []db.TokenVestingTable
		var vesting1, vesting2 db.TokenVestingTable
		var patches *gomonkey.Patches

		BeforeEach(func() {
			// Initialize user data
			tokenUserData = db.LogxTokenUserTable{
				UserAddress: "0x1234",
				TotalAmt:    ctypes.NewBigInt(big.NewInt(1000)),
				ClaimedAmt:  ctypes.NewBigInt(big.NewInt(0)),
				HasVesting:  true,
				IsBlocked:   false,
			}

			// Initialize first vesting data
			vesting1 = db.TokenVestingTable{
				UserAddress:          "0x1234",
				TotalAmt:             ctypes.NewBigInt(big.NewInt(1000)),
				ClaimedAmt:           ctypes.NewBigInt(big.NewInt(0)),
				AmtUnlockDay1:        ctypes.NewBigInt(big.NewInt(200)),
				UnlockPercentageX100: ctypes.NewBigInt(big.NewInt(333)), // 3.33% daily unlock
				Frequency:            1,                                 // Daily unlock
				VestingType:          "KOL",
			}

			// Initialize second vesting data
			vesting2 = db.TokenVestingTable{
				UserAddress:          "0x1234",
				TotalAmt:             ctypes.NewBigInt(big.NewInt(1000)),
				ClaimedAmt:           ctypes.NewBigInt(big.NewInt(0)),
				AmtUnlockDay1:        ctypes.NewBigInt(big.NewInt(100)),
				UnlockPercentageX100: ctypes.NewBigInt(big.NewInt(666)), // 6.66% daily unlock
				Frequency:            1,                                 // Daily unlock
				VestingType:          "DEAL",
			}

			// Combine both vesting entries for the user
			vestingData = []db.TokenVestingTable{vesting1, vesting2}

			// Initialize LogxTokenUtils
			tokenUtils = logxTokenUtils.NewLogxTokenUtils()
		})

		AfterEach(func() {
			if patches != nil {
				patches.Reset()
			}
		})

		// It("should update claimed amount across vesting records if claimable amount matches", func() {
		// 	// Mock ClaimableAmountifVesting to return a total claimable amount equal to the input amount
		// 	claimableAmount := ctypes.NewBigInt(big.NewInt(300)) // Simulate the claimable amount
		// 	vestingAmountMap := map[uint]ctypes.BigInt{
		// 		1: ctypes.NewBigInt(big.NewInt(200)), // vesting1
		// 		2: ctypes.NewBigInt(big.NewInt(100)), // vesting2
		// 	}

		// 	patches = gomonkey.ApplyMethod(reflect.TypeOf(tokenUtils), "ClaimableAmountifVesting", func(_ *logxTokenUtils.LogxTokenUtils, _ *db.LogxTokenUserTable, _ []db.TokenVestingTable, _ time.Time) (ctypes.BigInt, ctypes.BigInt, int64, ctypes.BigInt, map[uint]ctypes.BigInt, error) {
		// 		return claimableAmount, ctypes.NewBigInt(big.NewInt(1000)), 0, ctypes.NewBigInt(big.NewInt(0)), vestingAmountMap, nil
		// 	})

		// 	// Mock UpdateClaimedAmtInVestingTable to simulate successful database update
		// 	patches.ApplyMethod(reflect.TypeOf(tokenUtils.TokenDB), "UpdateClaimedAmtInVestingTable", func(_ *db.LogxTokenDB, userAddress string, vestingID uint, newClaimedAmt ctypes.BigInt) error {
		// 		return nil
		// 	})

		// 	// Simulate the input amount to be equal to the claimable amount
		// 	inputAmount := ctypes.NewBigInt(big.NewInt(300))

		// 	// Call UpdateClaimedAmtInVesting
		// 	err := tokenUtils.UpdateClaimedAmtInVesting("0x1234", vestingData, inputAmount, &tokenUserData)

		// 	// Expect no error
		// 	Expect(err).To(BeNil())
		// })

		It("should return an error if input amount does not match claimable amount", func() {
			// Mock ClaimableAmountifVesting to return a different claimable amount
			patches = gomonkey.ApplyMethod(reflect.TypeOf(tokenUtils), "ClaimableAmountifVesting", func(_ *logxTokenUtils.LogxTokenUtils, _ *db.LogxTokenUserTable, _ []db.TokenVestingTable, _ time.Time) (ctypes.BigInt, ctypes.BigInt, int64, ctypes.BigInt, map[uint]ctypes.BigInt, error) {
				return ctypes.NewBigInt(big.NewInt(200)), ctypes.NewBigInt(big.NewInt(1000)), 0, ctypes.NewBigInt(big.NewInt(0)), nil, nil
			})

			// Simulate the input amount to be different from the claimable amount
			inputAmount := ctypes.NewBigInt(big.NewInt(300))

			// Call UpdateClaimedAmtInVesting
			err := tokenUtils.UpdateClaimedAmtInVesting("0x1234", vestingData, inputAmount, &tokenUserData)

			// Expect an error
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("input amount does not match the total claimable amount"))
		})
	})

	/// There is some issue coming while patching need to fix

	// Describe("UpdateTokenUserDataByUserAddress", func() {
	// 	var tokenUtils *logxTokenUtils.LogxTokenUtils
	// 	var patches *gomonkey.Patches
	// 	var userAddress string
	// 	var amount ctypes.BigInt
	// 	var tokenUserData db.LogxTokenUserTable
	// 	var vestingData []db.TokenVestingTable

	// 	BeforeEach(func() {
	// 		// Initialize LogxTokenUtils
	// 		tokenUtils = logxTokenUtils.NewLogxTokenUtils()

	// 		// Initialize common variables
	// 		userAddress = "0x1234"
	// 		amount = ctypes.NewBigInt(big.NewInt(500))

	// 		// Initialize user data
	// 		tokenUserData = db.LogxTokenUserTable{
	// 			UserAddress: "0x1234",
	// 			TotalAmt:    ctypes.NewBigInt(big.NewInt(1000)),
	// 			ClaimedAmt:  ctypes.NewBigInt(big.NewInt(500)),
	// 			HasVesting:  false,
	// 			IsBlocked:   false,
	// 		}

	// 		// Initialize vesting data (for the cases where the user has vesting)
	// 		vestingData = []db.TokenVestingTable{
	// 			{
	// 				UserAddress:          "0x1234",
	// 				TotalAmt:             ctypes.NewBigInt(big.NewInt(1000)),
	// 				ClaimedAmt:           ctypes.NewBigInt(big.NewInt(0)),
	// 				AmtUnlockDay1:        ctypes.NewBigInt(big.NewInt(200)),
	// 				UnlockPercentageX100: ctypes.NewBigInt(big.NewInt(333)), // 3.33% daily unlock
	// 				Frequency:            1,                                 // Daily unlock
	// 				VestingType:          "KOL",
	// 			},
	// 		}
	// 	})

	// 	AfterEach(func() {
	// 		if patches != nil {
	// 			patches.Reset()
	// 		}
	// 	})

	// 	It("should return error if the user is blocked", func() {
	// 		// Mock GetLogxTokenUserByUserAddress to return a blocked user
	// 		patches = gomonkey.ApplyMethod(reflect.TypeOf(tokenUtils.TokenDB), "GetLogxTokenUserByUserAddress", func(_ *db.LogxTokenDB, userAddress string) (*db.LogxTokenUserTable, error) {
	// 			return &db.LogxTokenUserTable{
	// 				UserAddress: "0x1234",
	// 				IsBlocked:   true,
	// 			}, nil
	// 		})

	// 		// Call UpdateTokenUserDataByUserAddress
	// 		success, err := tokenUtils.UpdateTokenUserDataByUserAddress(userAddress, amount)

	// 		// Expect an error and false return value
	// 		Expect(success).To(BeFalse())
	// 		Expect(err).ToNot(BeNil())
	// 		Expect(err.Error()).To(ContainSubstring("user is blocked"))
	// 	})

	// 	It("should return error if the user is blocked by IP", func() {
	// 		// Mock GetLogxTokenUserByUserAddress to return a user who is not blocked
	// 		patches = gomonkey.ApplyMethod(reflect.TypeOf(tokenUtils.TokenDB), "GetLogxTokenUserByUserAddress", func(_ *db.LogxTokenDB, userAddress string) (*db.LogxTokenUserTable, error) {
	// 			return &tokenUserData, nil
	// 		})

	// 		// Mock IsIPBlocked to return true (user blocked by IP)
	// 		patches.ApplyMethod(reflect.TypeOf(tokenUtils), "IsIPBlocked", func(_ *logxTokenUtils.LogxTokenUtils, _ string) (bool, error) {
	// 			return true, nil
	// 		})

	// 		// Call UpdateTokenUserDataByUserAddress
	// 		success, err := tokenUtils.UpdateTokenUserDataByUserAddress(userAddress, amount)

	// 		// Expect an error and false return value
	// 		Expect(success).To(BeFalse())
	// 		Expect(err).ToNot(BeNil())
	// 		Expect(err.Error()).To(ContainSubstring("user is blocked by IP"))
	// 	})

	// 	It("should update vesting claimed amount if user has vesting", func() {
	// 		// Mock GetLogxTokenUserByUserAddress to return a user with vesting
	// 		patches = gomonkey.ApplyMethod(reflect.TypeOf(tokenUtils.TokenDB), "GetLogxTokenUserByUserAddress", func(_ *db.LogxTokenDB, userAddress string) (*db.LogxTokenUserTable, error) {
	// 			return &db.LogxTokenUserTable{
	// 				UserAddress: "0x1234",
	// 				HasVesting:  true,
	// 				TotalAmt:    ctypes.NewBigInt(big.NewInt(1000)),
	// 				ClaimedAmt:  ctypes.NewBigInt(big.NewInt(500)),
	// 				IsBlocked:   false,
	// 			}, nil
	// 		})

	// 		// Mock GetAllTokenVestingsByUserAddress to return vesting data
	// 		patches.ApplyMethod(reflect.TypeOf(tokenUtils.TokenDB), "GetAllTokenVestingsByUserAddress", func(_ *db.LogxTokenDB, userAddress string) ([]db.TokenVestingTable, error) {
	// 			return vestingData, nil
	// 		})

	// 		// Mock UpdateClaimedAmtInVesting to simulate a successful update
	// 		patches.ApplyMethod(reflect.TypeOf(tokenUtils), "UpdateClaimedAmtInVesting", func(_ *logxTokenUtils.LogxTokenUtils, _ string, _ []db.TokenVestingTable, _ ctypes.BigInt, _ *db.LogxTokenUserTable) error {
	// 			return nil
	// 		})

	// 		// Call UpdateTokenUserDataByUserAddress
	// 		success, err := tokenUtils.UpdateTokenUserDataByUserAddress(userAddress, amount)

	// 		// Expect no error and success to be true
	// 		Expect(success).To(BeTrue())
	// 		Expect(err).To(BeNil())
	// 	})

	// 	It("should update claimed amount in LogxTokenUserTable if user has no vesting", func() {
	// 		// Mock GetLogxTokenUserByUserAddress to return a user without vesting
	// 		patches = gomonkey.ApplyMethod(reflect.TypeOf(tokenUtils.TokenDB), "GetLogxTokenUserByUserAddress", func(_ *db.LogxTokenDB, userAddress string) (*db.LogxTokenUserTable, error) {
	// 			return &tokenUserData, nil
	// 		})

	// 		// Mock UpdateClaimedAmtInLogxTokenUserTable to simulate a successful update
	// 		patches.ApplyMethod(reflect.TypeOf(tokenUtils.TokenDB), "UpdateClaimedAmtInLogxTokenUserTable", func(_ *db.LogxTokenDB, userAddress string, amount ctypes.BigInt) error {
	// 			return nil
	// 		})

	// 		// Call UpdateTokenUserDataByUserAddress
	// 		success, err := tokenUtils.UpdateTokenUserDataByUserAddress(userAddress, amount)

	// 		// Expect no error and success to be true
	// 		Expect(success).To(BeTrue())
	// 		Expect(err).To(BeNil())
	// 	})

	// 	It("should return false if remaining amount does not match input amount", func() {
	// 		// Mock GetLogxTokenUserByUserAddress to return a user without vesting
	// 		patches = gomonkey.ApplyMethod(reflect.TypeOf(tokenUtils.TokenDB), "GetLogxTokenUserByUserAddress", func(_ *db.LogxTokenDB, userAddress string) (*db.LogxTokenUserTable, error) {
	// 			return &db.LogxTokenUserTable{
	// 				UserAddress: "0x1234",
	// 				TotalAmt:    ctypes.NewBigInt(big.NewInt(1000)),
	// 				ClaimedAmt:  ctypes.NewBigInt(big.NewInt(400)), // Difference is 600
	// 				HasVesting:  false,
	// 				IsBlocked:   false,
	// 			}, nil
	// 		})

	// 		// Call UpdateTokenUserDataByUserAddress with an amount that doesn't match the remaining claimable amount
	// 		incorrectAmount := ctypes.NewBigInt(big.NewInt(500))

	// 		success, err := tokenUtils.UpdateTokenUserDataByUserAddress(userAddress, incorrectAmount)

	// 		// Expect no error but success to be false
	// 		Expect(success).To(BeFalse())
	// 		Expect(err).To(BeNil())
	// 	})
	// })

	Describe("IsIPBlocked", func() {
		var tokenUtils *logxTokenUtils.LogxTokenUtils
		var patches *gomonkey.Patches
		var subaccountID string

		BeforeEach(func() {
			// Initialize the LogxTokenUtils
			tokenUtils = &logxTokenUtils.LogxTokenUtils{
				CountryCodeDB: &db.CountryCodeDB{}, // Ensure CountryCodeDB is initialized
			}

			// Subaccount ID for the tests
			subaccountID = "12345"
		})

		AfterEach(func() {
			if patches != nil {
				patches.Reset() // Reset patches after each test
			}
		})

		Context("When the country is restricted", func() {
			BeforeEach(func() {
				// Patch the GetCountryCodeBySubaccount method to return a restricted country code
				patches = gomonkey.ApplyMethod(reflect.TypeOf(tokenUtils.CountryCodeDB), "GetCountryCodeBySubaccount", func(_ *db.CountryCodeDB, subaccountID string) *db.IpTable {
					return &db.IpTable{CountryCode: "US"}
				})

				// Patch the UpdateIsBlockedStatus method to simulate a successful update
				patches.ApplyMethod(reflect.TypeOf(tokenUtils.TokenDB), "UpdateIsBlockedStatus", func(_ *db.LogxTokenDB, subaccountID string, isBlocked bool) error {
					return nil
				})
			})

			It("should return true and block the user if the country is restricted", func() {
				// Call the function
				isBlocked, err := tokenUtils.IsIPBlocked(subaccountID)

				// Expect no error
				Expect(err).To(BeNil())

				// Verify that the user is blocked
				Expect(isBlocked).To(BeTrue())
			})
		})

		Context("When the country is not restricted", func() {
			BeforeEach(func() {
				// Patch the GetCountryCodeBySubaccount method to return a non-restricted country code
				patches = gomonkey.ApplyMethod(reflect.TypeOf(tokenUtils.CountryCodeDB), "GetCountryCodeBySubaccount", func(_ *db.CountryCodeDB, subaccountID string) *db.IpTable {
					return &db.IpTable{CountryCode: "IN"}
				})

				// Patch the UpdateIsBlockedStatus method to simulate a successful update
				patches.ApplyMethod(reflect.TypeOf(tokenUtils.TokenDB), "UpdateIsBlockedStatus", func(_ *db.LogxTokenDB, subaccountID string, isBlocked bool) error {
					return nil
				})
			})

			It("should return false and not block the user if the country is not restricted", func() {
				// Call the function
				isBlocked, err := tokenUtils.IsIPBlocked(subaccountID)

				// Expect no error
				Expect(err).To(BeNil())

				// Verify that the user is not blocked
				Expect(isBlocked).To(BeFalse())
			})
		})

		// Context("When the country code is not found", func() {
		// 	BeforeEach(func() {
		// 		// Patch the GetCountryCodeBySubaccount method to return nil (no record found)
		// 		patches = gomonkey.ApplyMethod(reflect.TypeOf(tokenUtils.CountryCodeDB), "GetCountryCodeBySubaccount", func(_ *db.CountryCodeDB, subaccountID string) *db.IpTable {
		// 			return nil
		// 		})
		// 	})

		// 	It("should return an error if the country code is not found", func() {
		// 		// Call the function
		// 		isBlocked, err := tokenUtils.IsIPBlocked(subaccountID)

		// 		// Expect an error
		// 		Expect(err).ToNot(BeNil())
		// 		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("could not fetch country code for subaccount: %s", subaccountID)))

		// 		// Verify that the user is not blocked
		// 		Expect(isBlocked).To(BeFalse())
		// 	})
		// })

		Context("When updating IsBlocked status fails", func() {
			BeforeEach(func() {
				// Patch the GetCountryCodeBySubaccount method to return a restricted country code
				patches = gomonkey.ApplyMethod(reflect.TypeOf(tokenUtils.CountryCodeDB), "GetCountryCodeBySubaccount", func(_ *db.CountryCodeDB, subaccountID string) *db.IpTable {
					return &db.IpTable{CountryCode: "US"}
				})

				// Patch the UpdateIsBlockedStatus method to simulate a failure
				patches.ApplyMethod(reflect.TypeOf(tokenUtils.TokenDB), "UpdateIsBlockedStatus", func(_ *db.LogxTokenDB, subaccountID string, isBlocked bool) error {
					return errors.New("database update error")
				})
			})

			// It("should return an error if updating IsBlocked status fails", func() {
			// 	// Call the function
			// 	isBlocked, err := tokenUtils.IsIPBlocked(subaccountID)

			// 	// Expect an error
			// 	Expect(err).ToNot(BeNil())
			// 	Expect(err.Error()).To(ContainSubstring("failed to update IsBlocked status"))

			// 	// Verify that the user is not blocked
			// 	Expect(isBlocked).To(BeFalse())
			// })
		})
	})
})
