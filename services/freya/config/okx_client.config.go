package config

// we can fetch this dynamically via websocket in the future
// How to calculate lot size multiplier :
//
//	it is 1 / "ctVal" in the response of 'GET /api/v5/public/instruments?instType=SPOT' api on OKX
//
// more information - https://my.okx.com/docs-v5/en/#public-data-rest-api-get-instruments
var SYMBOL_TO_LOT_SIZE = map[string]float64{
	"ETH":  100,
	"BTC":  1000,
	"SOL":  10,
	"DOGE": 0.1,
	"ARB":  0.1,
	"LINK": 1,
	"XRP":  0.01,
	"NEAR": 0.1,
	// EIGEN and TON Perps do not exist on OKX DEMO Trading
	// "EIGEN": XX,
	// "TON":   XX,
}
