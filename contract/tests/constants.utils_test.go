package tests

import (
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/testutils"
	"slices"
	"sort"
	"testing"
)

func TestOrderOfAllSpots(t *testing.T) {
	// Should be in increasing order
	spotsCopy := make([]uint32, len(contractUtils.ALL_SPOTS_IN_ORDER))
	copy(spotsCopy, contractUtils.ALL_SPOTS_IN_ORDER)

	sort.Slice(spotsCopy, func(i, j int) bool {
		return spotsCopy[i] < spotsCopy[j]
	})

	if !slices.Equal(spotsCopy, contractUtils.ALL_SPOTS_IN_ORDER) {
		t.Errorf("The order of spots is not correct")
	}
}

func TestOrderOfAllPerpMarkets(t *testing.T) {
	// Should be in increasing order
	testutils.SetMainnetEnv()
	perpMarketsCopy := make([]uint32, len(contractUtils.ALL_PERPS_ON_CONTRACT))

	copy(perpMarketsCopy, contractUtils.ALL_PERPS_ON_CONTRACT)

	sort.Slice(perpMarketsCopy, func(i, j int) bool {
		return perpMarketsCopy[i] < perpMarketsCopy[j]
	})

	if !slices.Equal(perpMarketsCopy, contractUtils.ALL_PERPS_ON_CONTRACT) {
		t.Errorf("The order of perp markets is not correct")
	}
}
