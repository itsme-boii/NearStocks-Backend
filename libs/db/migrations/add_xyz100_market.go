package migrations

import (
	"fmt"

	"github/eugenix-io/logx-inf-backend/libs/db"
)

// AddXYZ100Market creates the XYZ100 market entry in the database
// Run this migration to add the XYZ100-USD perpetual market
//
// Usage: Call this function after db.Init() has been called
//
// Market Configuration:
//   - ID: 223 (matching XYZ100_MARKET constant in constants.utils.go)
//   - Symbol: XYZ100-USD
//   - Type: PERPETUAL
//   - 24/7 trading (not in US_MARKET_IDS)
//   - 5% liquidation fraction
func AddXYZ100Market() error {
	// Using raw SQL to insert with specific ID
	// XYZ100 is a Nasdaq 100 tracking index on Hyperliquid (from trade.xyz)
	query := `
		INSERT INTO market_tables (
			id,
			created_at,
			updated_at,
			symbol,
			type,
			amt_to_qtm_conversion_expo,
			price_to_qtm_conversion_expo,
			base_asset,
			quote_asset,
			is_active,
			maker_fee_fractionx18,
			taker_fee_fractionx18,
			initial_margin_fractionx18,
			maintenance_margin_fractionx18,
			min_amountx18,
			max_position_valuex18
		) VALUES (
			223,
			NOW(),
			NOW(),
			'XYZ100-USD',
			'PERPETUAL',
			-9,
			-9,
			'XYZ100',
			'USDC',
			true,
			'0',
			'500000000000000',
			'50000000000000000',
			'25000000000000000',
			'1000000000000000000',
			'1000000000000000000000000'
		)
		ON CONFLICT (id) DO NOTHING
	`

	err := db.ExecSQL(query)
	if err != nil {
		return fmt.Errorf("failed to create XYZ100 market: %w", err)
	}

	fmt.Println("Successfully created XYZ100 market with ID: 223")
	return nil
}

// GetXYZ100MarketSQL returns the raw SQL for manual execution
// Use this if you need to run the migration outside of Go
func GetXYZ100MarketSQL() string {
	return `
INSERT INTO market_tables (
	id,
	created_at,
	updated_at,
	symbol,
	type,
	amt_to_qtm_conversion_expo,
	price_to_qtm_conversion_expo,
	base_asset,
	quote_asset,
	is_active,
	maker_fee_fractionx18,
	taker_fee_fractionx18,
	initial_margin_fractionx18,
	maintenance_margin_fractionx18,
	min_amountx18,
	max_position_valuex18
) VALUES (
	223,
	NOW(),
	NOW(),
	'XYZ100-USD',
	'PERPETUAL',
	-9,
	-9,
	'XYZ100',
	'USDC',
	true,
	'0',
	'500000000000000',
	'50000000000000000',
	'25000000000000000',
	'1000000000000000000',
	'1000000000000000000000000'
)
ON CONFLICT (id) DO NOTHING;
`
}
