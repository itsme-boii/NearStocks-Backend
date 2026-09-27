package contractUtils

import (
	"log"
	"os"
)

const (
	SESSION_KEY_CHAIN_ID                    = 1 // ETH
	QUOTE_TOKEN_PRODUCT_ID                  = 4
	BROKER_2_QUOTE_TOKEN_PRODUCT_ID         = 72 // OSTRICH QUOTE TOKEN for broker 2
	BROKER_2_UPDATED_QUOTE_TOKEN_PRODUCT_ID = 74
	LOGX                                    = 0
	ST_LOGX                                 = 2
	ETH_MARKET                              = 1
	BTC_MARKET                              = 3
	SOL_MARKET                              = 5
	PEPE_MARKET                             = 7
	DOGE_MARKET                             = 9
	TRUMP_MARKET                            = 11
	HARRIS_MARKET                           = 13
	ARB_MARKET                              = 15
	LINK_MARKET                             = 17
	XRP_MARKET                              = 19
	NEAR_MARKET                             = 21
	EIGEN_MARKET                            = 23
	TON_MARKET                              = 25
	GOAT_MARKET                             = 27
	SPX_MARKET                              = 29
	SCR_MARKET                              = 31
	WIF_MARKET                              = 33
	POPCAT_MARKET                           = 35
	NEIRO_MARKET                            = 37
	kPEPE_MARKET                            = 39
	kSHIB_MARKET                            = 41
	FARTCOIN_MARKET                         = 43
	MOODENG_MARKET                          = 45
	GIGA_MARKET                             = 47
	kBONK_MARKET                            = 49
	BRETT_MARKET                            = 51
	GRASS_MARKET                            = 53
	TIA_MARKET                              = 55
	SEI_MARKET                              = 57
	SUI_MARKET                              = 59
	ZRO_MARKET                              = 61
	TAIKO_MARKET                            = 63
	VIRTUAL_MARKET                          = 65
	BOME_MARKET                             = 67
	APE_MARKET                              = 69
	PONKE_MARKET                            = 71
	NPC_MARKET                              = 73
	SWELL_MARKET                            = 75
	JPY_MARKET                              = 77
	ALGO_MARKET                             = 79
	XLM_MARKET                              = 81
	HBAR_MARKET                             = 83
	EUR_MARKET                              = 85
	XAUT_MARKET                             = 87
	MOVE_MARKET                             = 89
	LOGX_MARKET                             = 91
	OFFICIAL_TRUMP_MARKET                   = 93
	WAL_MARKET                              = 95
	FLUID_MARKET                            = 97
	PROMPT_MARKET                           = 99
	NXPC_MARKET                             = 101
	TSLA_MARKET                             = 103
	NVDA_MARKET                             = 105
	AAPL_MARKET                             = 107
	QQQ_MARKET                              = 109
	BTC_OSTRICH_MARKET                      = 111
	ETH_OSTRICH_MARKET                      = 113
	SOL_OSTRICH_MARKET                      = 115
	XAUT_OSTRICH_MARKET                     = 117
	JPY_OSTRICH_MARKET                      = 119
	EUR_OSTRICH_MARKET                      = 121
	GBP_OSTRICH_MARKET                      = 123
	COIN_OSTRICH_MARKET                     = 125
	GOOGL_OSTRICH_MARKET                    = 127
	MSFT_OSTRICH_MARKET                     = 129
	AMZN_OSTRICH_MARKET                     = 131
	META_OSTRICH_MARKET                     = 133
	MSTR_OSTRICH_MARKET                     = 135
	DEFT_OSTRICH_MARKET                     = 137
	PLTR_OSTRICH_MARKET                     = 139
	HOOD_OSTRICH_MARKET                     = 141
	PYPL_OSTRICH_MARKET                     = 143
	GME_OSTRICH_MARKET                      = 145
	RIOT_OSTRICH_MARKET                     = 147
	COKE_OSTRICH_MARKET                     = 149
	CRCL_OSTRICH_MARKET                     = 151
	RDDT_OSTRICH_MARKET                     = 153
	USOIL_OSTRICH_MARKET                    = 155
	COPPER_OSTRICH_MARKET                   = 157
	NGAS_OSTRICH_MARKET                     = 159
	Silver_OSTRICH_MARKET                   = 161
	HIMS_OSTRICH_MARKET                     = 163
	SPY_OSTRICH_MARKET                      = 165
	NA_OSTRICH_MARKET                       = 167
	HUT_OSTRICH_MARKET                      = 169
	BRK_B_OSTRICH_MARKET                    = 171
	TCEHY_OSTRICH_MARKET                    = 173
	BIDU_OSTRICH_MARKET                     = 175
	BABA_OSTRICH_MARKET                     = 177
	NKE_OSTRICH_MARKET                      = 179
	SOFI_OSTRICH_MARKET                     = 181
	PUMP_MARKET                             = 183
	NIO_OSTRICH_MARKET                      = 185
	JD_OSTRICH_MARKET                       = 187
	TRY_OSTRICH_MARKET                      = 189
	AUD_OSTRICH_MARKET                      = 191
	NZD_OSTRICH_MARKET                      = 193
	CHF_OSTRICH_MARKET                      = 195
	YHC_OSTRICH_MARKET                      = 197
	TRON_OSTRICH_MARKET                     = 199
	SONY_OSTRICH_MARKET                     = 201
	MTPLF_OSTRICH_MARKET                    = 203
	MUFG_OSTRICH_MARKET                     = 205
	TM_OSTRICH_MARKET                       = 207
	HMC_OSTRICH_MARKET                      = 209
	AZN_OSTRICH_MARKET                      = 211
	HSBC_OSTRICH_MARKET                     = 213
	SHEL_OSTRICH_MARKET                     = 215
	BCS_OSTRICH_MARKET                      = 217
	RYCEY_OSTRICH_MARKET                    = 219
	RICH_OSTRICH_MARKET                     = 221
	XYZ100_MARKET                           = 223
	AMM_SUBACCOUNT_ID                       = "0x0000000000010000000000000000000000000000000000000001000000000001"
	AMM_SUBACCOUNT_ID_1                     = "1_0x0000000000000000000000000000000000000001_1"
	LIQUIDATION_SUBACCOUNT_ID               = "0x0000000000010000000000000000000000000000000000000004000000000001"
	INSURANCE_SUBACCOUNT_ID                 = "0x0000000000010000000000000000000000000000000000000005000000000001"
	CAMPAIGN_REWARD_SUBACCOUNT_ID           = "0x0000000000010000000000000000000000000000000000000007000000000001"
	OPTIONS_X_SUBACCOUNT_ID                 = "0x0000000000010000000000000000000000000000000000000008000000000001"
	OPTIONS_FEES_SUBACCOUNT_ID              = "0x0000000000010000000000000000000000000000000000000009000000000001"
	PRE_MARKETS_X_SUBACCOUNT_ID             = "0x000000000001000000000000000000000000000000000000000b000000000001"
	PRE_MARKETS_FEES_SUBACCOUNT_ID          = "0x000000000001000000000000000000000000000000000000000c000000000001"
	SYN_SPOTS_X_SUBACCOUNT_ID               = "0x000000000001000000000000000000000000000000000000000d000000000001"
	SYN_SPOTS_FEES_SUBACCOUNT_ID            = "0x000000000001000000000000000000000000000000000000000e000000000001"
	TRADING_FEES_SUBACCOUNT_ID              = "0x000000000001000000000000000000000000000000000000000f000000000001" // D-7: trading and withdrawal fees, so balances reconcile exactly with custody
	LOGX_REWARDS_SUBACCOUNT_ID              = "0x0000000000010000000000000000000000000000000000000010000000000001" // NEAR: reward and airdrop claims are paid from this DAO-funded pool
	TESTNET                                 = "TESTNET"
	MAINNET                                 = "MAINNET"
	REFERRER_REWARD_ID                      = 4
	REFERRE_REWARD_ID                       = 4
	REFERRER_REBATE_PERCENTAGE              = 10
	REFERRAL_CAMPAIGN_ID                    = 1
	LOGX_FEE_REWARDS_CAMPAIGN_ID            = 2
	SETTLE_PNL_BATCH_SIZE                   = 100
)

// US_MARKET_IDS gates orders to real NYSE/Nasdaq hours (order.controller.go, when ENV=MAINNET and
// the MARKET_ORDERS_ENABLED Redis flag is "1"). near-stocks's listed markets (TSLA/NVDA/AAPL/QQQ/
// SPY/COIN/GOOGL/MSFT/AMZN/META/MSTR/PLTR/HOOD/CRCL, Development.md §16.8) are deliberately NOT in
// this map: the product is 24/7 synthetic exposure, unlike the legacy Ostrich spot product this map
// was built for, so they must never be hours-gated regardless of that flag's value in any
// environment.
var US_MARKET_IDS = map[uint]bool{
	DEFT_OSTRICH_MARKET:  true,
	PYPL_OSTRICH_MARKET:  true,
	GME_OSTRICH_MARKET:   true,
	RIOT_OSTRICH_MARKET:  true,
	COKE_OSTRICH_MARKET:  true,
	RDDT_OSTRICH_MARKET:  true,
	HIMS_OSTRICH_MARKET:  true,
	NA_OSTRICH_MARKET:    true,
	HUT_OSTRICH_MARKET:   true,
	BRK_B_OSTRICH_MARKET: true,
	TCEHY_OSTRICH_MARKET: true,
	BIDU_OSTRICH_MARKET:  true,
	BABA_OSTRICH_MARKET:  true,
	NKE_OSTRICH_MARKET:   true,
	SOFI_OSTRICH_MARKET:  true,
	NIO_OSTRICH_MARKET:   true,
	JD_OSTRICH_MARKET:    true,
	TRY_OSTRICH_MARKET:   true,
	AUD_OSTRICH_MARKET:   true,
	NZD_OSTRICH_MARKET:   true,
	CHF_OSTRICH_MARKET:   true,
	YHC_OSTRICH_MARKET:   true,
	TRON_OSTRICH_MARKET:  true,
	SONY_OSTRICH_MARKET:  true,
	MTPLF_OSTRICH_MARKET: true,
	MUFG_OSTRICH_MARKET:  true,
	TM_OSTRICH_MARKET:    true,
	HMC_OSTRICH_MARKET:   true,
	AZN_OSTRICH_MARKET:   true,
	HSBC_OSTRICH_MARKET:  true,
	SHEL_OSTRICH_MARKET:  true,
	BCS_OSTRICH_MARKET:   true,
	RYCEY_OSTRICH_MARKET: true,
}

type SourceChainInfo struct {
	MailboxAddress string
	RPC_URL        string
}

type Chain struct {
	Name      string
	Rpc       string
	NativeId  string
	Threshold float64
}

var (
	RPC_URL                         string
	ENDPOINT_CONTRACT_ADDRESS       string
	OFFCHAIN_CONTRACT_ADDRESS       string
	STAKER_CONTRACT_ADDRESS         string
	BRIDGE_OUT_CONTRACT             string
	CLEARING_HOUSE                  string
	PERP_CONTRACT_ADDRESS           string
	SPOT_CONTRACT_ADDRESS           string
	SOURCE_HYPERC_ADDRESS_MAP       map[uint]string // PRODUCT_ID -> HYPERC_ADDRESS
	ARB_USDT                        uint32          /***** MAINNET VARS START *******/
	ARB_USDC                        uint32          /***** MAINNET VARS END *******/
	ETH_USDT                        uint32          /***** TESTNET ONLY VARS START *******/
	ETH_USDC                        uint32
	OP_USDC                         uint32
	OP_USDT                         uint32
	BASE_USDC                       uint32
	BASE_USDT                       uint32
	MANTLE_USDC                     uint32
	MANTLE_USDT                     uint32
	BERACHAIN_USDC                  uint32
	BERACHAIN_USDT                  uint32
	MODE_USDC                       uint32
	MODE_USDT                       uint32
	KROMA_USDC                      uint32
	KROMA_USDT                      uint32
	MONADTESTNET_USDC               uint32 /***** TESTNET ONLY VARS END *******/
	BOB_USDC                        uint32
	BOB_USDT                        uint32
	SEI_USDC                        uint32
	SEI_USDT                        uint32
	SCROLL_USDC                     uint32
	SCROLL_USDT                     uint32
	TAIKO_USDC                      uint32
	TAIKO_USDT                      uint32
	LINEA_USDC                      uint32
	LINEA_USDT                      uint32
	RARI_USDC                       uint32
	MINT_USDC                       uint32
	MINT_USDT                       uint32
	ABSTRACT_USDC                   uint32
	APECHAIN_USD                    uint32
	APECHAIN_USDC                   uint32
	BLAST_USDB                      uint32
	OSTRICH_USDC                    uint32
	OSTRICH_USDC_UPDATED            uint32
	SONIC_USDC                      uint32
	SONIC_scUSD                     uint32
	MANTA_USDC                      uint32
	MANTA_wUSDM                     uint32
	POLYGON_USDC                    uint32
	POLYGON_USDT                    uint32
	PRODUCT_ID_SYMBOL_TO_MAP        map[uint32]string
	PRODUCT_MARKET_WEIGHTS          map[uint32]int
	SPOT_TO_PRE_MARKET_MAP          map[uint32]uint32
	PRE_MARKET_TO_SPOT_FACTOR_MAP   map[uint32]uint32
	ALL_SPOTS_IN_ORDER              []uint32
	ALL_SPOTS_ON_CONTRACT           []uint32
	ALL_COLLATERAL_SPOTS            []uint32
	ALL_SPOTS_FOR_INSURANCE         []uint32
	ALL_NON_PREDICTION_PERPS        []uint32
	ALL_PREDICTION_PERPS            []uint32
	ALL_MEME_PERPS                  []uint32
	ALL_PERPS_ON_CONTRACT           []uint32
	ALL_COLLATERAL_TOKEN_SYMBOLS    []string
	ARBITRUM                        int64
	ETHEREUM                        int64
	OPTIMISM                        int64
	BASE                            int64
	MANTLE                          int64
	MODE                            int64
	BERACHAIN                       int64
	MONADTESTNET                    int64
	BOB                             int64
	SEI                             int64
	SCROLL                          int64
	KROMA                           int64
	TAIKO                           int64
	LINEA                           int64
	RARI                            int64
	MINT                            int64
	ABSTRACT                        int64
	APECHAIN                        int64
	SONIC                           int64
	MANTA                           int64
	BLAST                           int64
	POLYGON                         int64
	ProductChainMapping             map[uint32]int64
	SUPPORTED_LOGX_CHAINS           map[int64]bool
	PRODUCT_ID_TO_SOURCE_CHAIN_INFO map[uint32]SourceChainInfo
	LOGX_CHAIN_ID                   int64
	ARBITRUM_CHAIN_ID               int64
	ETHEREUM_CHAIN_ID               int64
	LOGX_MAILBOX_ADDRESS            string
	OSTRICH_MAILBOX_ADDRESS         string
	RELAYER_CHAINS                  []Chain
	LIQUIDATION_FRACTION_STR        map[uint32]string
	OPTIONS_INTERVALS               map[uint32]bool
	SYMBOL_TO_OSTRICH_SYMBOL        map[string]string // Map base symbols to their Ostrich market symbols
	WITHDRAWAL_FEE_MAP              map[uint32]string
	PRODUCT_ID_TO_MAILBOX_ADDRESS   map[uint32]string
)

func Init() {
	currentEnv := os.Getenv("ENV")
	switch currentEnv {
	case TESTNET:
		RPC_URL = "" // tokenised URL — must come from LOGX_RPC_URL
		ENDPOINT_CONTRACT_ADDRESS = "0x15B170A89C2cD06825de4D6Fc1A60dDEb03a06ba"
		OFFCHAIN_CONTRACT_ADDRESS = "0x80C9af965c0a6B16089C8f049a386053548E97b1"
		STAKER_CONTRACT_ADDRESS = "0x43A99F607567320892395B7fcfA25e79E33De03e"
		BRIDGE_OUT_CONTRACT = "0x83763e516D60b18baE8294E0B8ea0ab8865aeB23" // This is from kartel needs to be updated
		CLEARING_HOUSE = "0x03C33476a438c3D1402dfc96F5b832bd5917e964"
		PERP_CONTRACT_ADDRESS = "0x5C5D91065883d6Ac3e862b1Fabb9899CDfe61c61"
		SPOT_CONTRACT_ADDRESS = "0xCe7E2833C7435C6689c425851F640e8000a112Af"
		LOGX_MAILBOX_ADDRESS = "0xEBd5303C8A319bfc5bcb97E22a92834385E697B0" // This is from kartel needs to be updated
		OSTRICH_MAILBOX_ADDRESS = "0x915e9e1E4E1b5842597d799d7920c76Afd5408DC"
		ETH_USDC = 4
		ARB_USDC = 6
		ETH_USDT = 8
		ARB_USDT = 10
		OP_USDC = 12
		OP_USDT = 14
		BASE_USDC = 16
		BASE_USDT = 18
		MANTLE_USDC = 20
		MANTLE_USDT = 22
		BERACHAIN_USDC = 24
		BERACHAIN_USDT = 26
		MODE_USDC = 28
		MODE_USDT = 30
		KROMA_USDC = 32
		KROMA_USDT = 34
		POLYGON_USDC = 52
		POLYGON_USDT = 54
		MONADTESTNET_USDC = 36
		LOGX_CHAIN_ID = 116854
		ARBITRUM_CHAIN_ID = 421614
		ETHEREUM_CHAIN_ID = 11155111

		LIQUIDATION_FRACTION_STR = map[uint32]string{
			GOAT_MARKET:           "0.05", // 5%
			SPX_MARKET:            "0.05", // 5%
			WIF_MARKET:            "0.05", // 5%
			POPCAT_MARKET:         "0.05", // 5%
			NEIRO_MARKET:          "0.05", // 5%
			kPEPE_MARKET:          "0.05", // 5%
			kSHIB_MARKET:          "0.05", // 5%
			FARTCOIN_MARKET:       "0.05", // 5%
			MOODENG_MARKET:        "0.05", // 5%
			GIGA_MARKET:           "0.05", // 5%
			kBONK_MARKET:          "0.05", // 5%
			BRETT_MARKET:          "0.05", // 5%
			GRASS_MARKET:          "0.05", // 5%
			BOME_MARKET:           "0.05", // 5%
			APE_MARKET:            "0.05", // 5%
			PONKE_MARKET:          "0.05", // 5%
			NPC_MARKET:            "0.05", // 5%
			OFFICIAL_TRUMP_MARKET: "0.10", // 10%
			WAL_MARKET:            "0.03", // 3%
			FLUID_MARKET:          "0.04", // 4%
			PROMPT_MARKET:         "0.04", // 4%
			NXPC_MARKET:           "0.04", // 4%
			TSLA_MARKET:           "0.02", // 2%
			NVDA_MARKET:           "0.02", // 2%
			AAPL_MARKET:           "0.02", // 2%
			QQQ_MARKET:            "0.02", // 2%
			BTC_OSTRICH_MARKET:    "0.01", // 1%
			ETH_OSTRICH_MARKET:    "0.01", // 1%
			SOL_OSTRICH_MARKET:    "0.01", // 1%
			XAUT_OSTRICH_MARKET:   "0.02", // 2%
			JPY_OSTRICH_MARKET:    "0.02", // 2%
			EUR_OSTRICH_MARKET:    "0.02", // 2%
			GBP_OSTRICH_MARKET:    "0.02", // 2%
			COIN_OSTRICH_MARKET:   "0.02", // 2%
			GOOGL_OSTRICH_MARKET:  "0.02", // 2%
			MSFT_OSTRICH_MARKET:   "0.02", // 2%
			AMZN_OSTRICH_MARKET:   "0.02", // 2%
			META_OSTRICH_MARKET:   "0.02", // 2%
			MSTR_OSTRICH_MARKET:   "0.02", // 2%
			DEFT_OSTRICH_MARKET:   "0.02", // 2%
			PLTR_OSTRICH_MARKET:   "0.02", // 2%
			HOOD_OSTRICH_MARKET:   "0.02", // 2%
			PYPL_OSTRICH_MARKET:   "0.02", // 2%
			GME_OSTRICH_MARKET:    "0.02", // 2%
			RIOT_OSTRICH_MARKET:   "0.02", // 2%
			COKE_OSTRICH_MARKET:   "0.02", // 2%
			CRCL_OSTRICH_MARKET:   "0.02", // 2%
			RDDT_OSTRICH_MARKET:   "0.02", // 2%
			USOIL_OSTRICH_MARKET:  "0.02", // 2%
			COPPER_OSTRICH_MARKET: "0.02", // 2%
			NGAS_OSTRICH_MARKET:   "0.02", // 2%
			Silver_OSTRICH_MARKET: "0.02", // 2%
			HIMS_OSTRICH_MARKET:   "0.02", // 2%
			SPY_OSTRICH_MARKET:    "0.02", // 2%
			NA_OSTRICH_MARKET:     "0.02", // 2%
			HUT_OSTRICH_MARKET:    "0.02", // 2%
			BRK_B_OSTRICH_MARKET:  "0.02", // 2%
			TCEHY_OSTRICH_MARKET:  "0.02", // 2%
			BIDU_OSTRICH_MARKET:   "0.02", // 2%
			BABA_OSTRICH_MARKET:   "0.02", // 2%
			NKE_OSTRICH_MARKET:    "0.02", // 2%
			SOFI_OSTRICH_MARKET:   "0.02", // 2%
			PUMP_MARKET:           "0.05", // 5%
			XYZ100_MARKET:         "0.05", // 5%
			NIO_OSTRICH_MARKET:    "0.02", // 2%
			JD_OSTRICH_MARKET:     "0.02", // 2%
			TRY_OSTRICH_MARKET:    "0.02", // 2%
			AUD_OSTRICH_MARKET:    "0.02", // 2%
			NZD_OSTRICH_MARKET:    "0.02", // 2%
			CHF_OSTRICH_MARKET:    "0.02", // 2%
			YHC_OSTRICH_MARKET:    "0.05", // 2%
			TRON_OSTRICH_MARKET:   "0.05", // 5%
			SONY_OSTRICH_MARKET:   "0.02", // 2%
			MTPLF_OSTRICH_MARKET:  "0.02", // 2%
			MUFG_OSTRICH_MARKET:   "0.02", // 2%
			TM_OSTRICH_MARKET:     "0.02", // 2%
			HMC_OSTRICH_MARKET:    "0.02", // 2%
			AZN_OSTRICH_MARKET:    "0.02", // 2%
			HSBC_OSTRICH_MARKET:   "0.02", // 2%
			SHEL_OSTRICH_MARKET:   "0.02", // 2%
			BCS_OSTRICH_MARKET:    "0.02", // 2%
			RYCEY_OSTRICH_MARKET:  "0.02", // 2%
			RICH_OSTRICH_MARKET:   "0.02", // 2%
		}
		SOURCE_HYPERC_ADDRESS_MAP = map[uint]string{
			0:  "0xDE1e3e11122dB0B2Fb24d12B32C163DD0c5Ade31",
			4:  "0x65F494A92DdC4fA10Ee8Eda9ac4B3C1105e3088C",
			6:  "0xe6410826e5E1b2f56aaB56feA04af41dF3D7A973",
			8:  "0x93a875EE6862F8d650B935878D0A73E19A18b89d",
			10: "0xdaB099fe1664748933211D36C305E617332835ee",
			28: "0x9E3a8cf60A1e0FB816Fa3071B898F28Fd92Cc14c",
			30: "0xf7e514b18A365FE687F5B4618858caF5fa106Cc7",
			36: "0x80abb8a8Cd29912194dcA1c1f2fc8587bAD5B8D1",
		}

		PRODUCT_ID_SYMBOL_TO_MAP = map[uint32]string{
			0:     "LOGX",
			2:     "stLogX",
			4:     "USDC",
			6:     "USDC",
			8:     "USDT",
			10:    "USDT",
			12:    "USDC",
			14:    "USDT",
			16:    "USDC",
			18:    "USDT",
			20:    "USDC",
			22:    "USDT",
			24:    "USDC",
			26:    "USDT",
			28:    "USDC",
			30:    "USDT",
			32:    "USDC",
			34:    "USDT",
			36:    "USDC",
			52:    "USDC",
			54:    "USDT",
			1:     "ETH",
			3:     "BTC",
			5:     "SOL",
			7:     "PEPE",
			9:     "DOGE",
			11:    "XTRUMP",
			13:    "HARRIS",
			15:    "ARB",
			17:    "LINK",
			19:    "XRP",
			21:    "NEAR",
			23:    "EIGEN",
			25:    "TON",
			27:    "GOAT",
			29:    "SPX",
			31:    "SCR",
			33:    "WIF",
			35:    "POPCAT",
			37:    "NEIRO",
			39:    "kPEPE",
			41:    "kSHIB",
			43:    "FARTCOIN",
			45:    "MOODENG",
			47:    "GIGA",
			49:    "kBONK",
			51:    "BRETT",
			53:    "GRASS",
			55:    "TIA",
			57:    "SEI",
			59:    "SUI",
			61:    "ZRO",
			63:    "TAIKO",
			65:    "VIRTUAL",
			67:    "BOME",
			69:    "APE",
			71:    "PONKE",
			73:    "NPC",
			75:    "SWELL",
			77:    "JPY",
			79:    "ALGO",
			81:    "XLM",
			83:    "HBAR",
			85:    "EUR",
			87:    "XAUT",
			89:    "MOVE",
			91:    "LOGX",
			93:    "TRUMP",
			95:    "WAL",
			97:    "FLUID",
			99:    "PROMPT",
			101:   "NXPC",
			103:   "TSLA",
			105:   "NVDA",
			107:   "AAPL",
			109:   "QQQ",
			111:   "BTC_OSTRICH",
			113:   "ETH_OSTRICH",
			115:   "SOL_OSTRICH",
			117:   "XAUT_OSTRICH",
			119:   "JPY_OSTRICH",
			121:   "EUR_OSTRICH",
			123:   "GBP_OSTRICH",
			125:   "COIN_OSTRICH",
			127:   "GOOGL_OSTRICH",
			129:   "MSFT_OSTRICH",
			131:   "AMZN_OSTRICH",
			133:   "META_OSTRICH",
			135:   "MSTR_OSTRICH",
			137:   "DEFT_OSTRICH",
			139:   "PLTR_OSTRICH",
			141:   "HOOD_OSTRICH",
			143:   "PYPL_OSTRICH",
			145:   "GME_OSTRICH",
			147:   "RIOT_OSTRICH",
			149:   "COKE_OSTRICH",
			151:   "CRCL_OSTRICH",
			153:   "RDDT_OSTRICH",
			155:   "USOIL_OSTRICH",
			157:   "COPPER_OSTRICH",
			159:   "NGAS_OSTRICH",
			161:   "Silver_OSTRICH",
			163:   "HIMS_OSTRICH",
			165:   "SPY_OSTRICH",
			167:   "NA_OSTRICH",
			169:   "HUT_OSTRICH",
			171:   "BRK_B_OSTRICH",
			173:   "TCEHY_OSTRICH",
			175:   "BIDU_OSTRICH",
			177:   "BABA_OSTRICH",
			179:   "NKE_OSTRICH",
			181:   "SOFI_OSTRICH",
			183:   "PUMP",
			185:   "NIO_OSTRICH",
			187:   "JD_OSTRICH",
			189:   "TRY_OSTRICH",
			191:   "AUD_OSTRICH",
			193:   "NZD_OSTRICH",
			195:   "CHF_OSTRICH",
			197:   "YHC_OSTRICH",
			199:   "TRON_OSTRICH",
			201:   "SONY_OSTRICH",
			203:   "MTPLF_OSTRICH",
			205:   "MUFG_OSTRICH",
			207:   "TM_OSTRICH",
			209:   "HMC_OSTRICH",
			211:   "AZN_OSTRICH",
			213:   "HSBC_OSTRICH",
			215:   "SHEL_OSTRICH",
			217:   "BCS_OSTRICH",
			219:   "RYCEY_OSTRICH",
			221:   "RICH_OSTRICH",
			223:   "XYZ100",
			20001: "BERA",
			20002: "LOGX",
			20003: "ETH",
			20004: "BTC",
			20005: "IP",
			20006: "KAITO",
			20007: "RED",
			20008: "ELX",
			20009: "MUBARAK",
			20010: "K",
			20011: "INIT",
			10001: "ETH",
			10003: "BTC",
			10005: "SOL",
			10007: "HYPE",
		}
		SPOT_TO_PRE_MARKET_MAP = map[uint32]uint32{
			20001: 1001,
			20005: 1003,
			20006: 1008,
			20007: 1004,
			20010: 1012,
			20011: 1014,
		}
		PRE_MARKET_TO_SPOT_FACTOR_MAP = map[uint32]uint32{
			1001: 50,
			1003: 100,
			1008: 100,
			1004: 100,
			1012: 1,
			1014: 100,
		}
		PRODUCT_MARKET_WEIGHTS = map[uint32]int{
			LOGX:              0, //0
			ST_LOGX:           0, //2
			ETH_USDC:          1, //4
			ARB_USDC:          1, //6
			ETH_USDT:          1, //8
			ARB_USDT:          1, //10
			OP_USDC:           1, //12
			OP_USDT:           1, //14
			BASE_USDC:         1, //16
			BASE_USDT:         1, //18
			MANTLE_USDC:       1, //20
			MANTLE_USDT:       1, //22
			BERACHAIN_USDC:    1, //24
			BERACHAIN_USDT:    1, //26
			MODE_USDC:         1, //28
			MODE_USDT:         1, //30
			KROMA_USDC:        1, //32
			KROMA_USDT:        1, //34
			MONADTESTNET_USDC: 1, //36
		}
		ALL_SPOTS_IN_ORDER = []uint32{
			LOGX,
			ST_LOGX,
			ETH_USDC,
			ARB_USDC,
			ETH_USDT,
			ARB_USDT,
			OP_USDC,
			OP_USDT,
			BASE_USDC,
			BASE_USDT,
			MANTLE_USDC,
			MANTLE_USDT,
			BERACHAIN_USDC,
			BERACHAIN_USDT,
			MODE_USDC,
			MODE_USDT,
			KROMA_USDC,
			KROMA_USDT,
			MONADTESTNET_USDC,
		}
		ALL_SPOTS_ON_CONTRACT = []uint32{
			ETH_USDC,
			LOGX,
			ST_LOGX,
			ARB_USDC,
			ETH_USDT,
			ARB_USDT,
			OP_USDC,
			OP_USDT,
			BASE_USDC,
			BASE_USDT,
			MANTLE_USDC,
			MANTLE_USDT,
			BERACHAIN_USDC,
			BERACHAIN_USDT,
			MODE_USDC,
			MODE_USDT,
			KROMA_USDC,
			KROMA_USDT,
			POLYGON_USDC,
			POLYGON_USDT,
			MONADTESTNET_USDC,
		}
		ALL_COLLATERAL_SPOTS = []uint32{
			ETH_USDC,
			ARB_USDC,
			ETH_USDT,
			ARB_USDT,
			OP_USDC,
			OP_USDT,
			BASE_USDC,
			BASE_USDT,
			MANTLE_USDC,
			MANTLE_USDT,
			BERACHAIN_USDC,
			BERACHAIN_USDT,
			MODE_USDC,
			MODE_USDT,
			KROMA_USDC,
			KROMA_USDT,
			MONADTESTNET_USDC,
		}
		// Do not include LogX, XLogX and QUOTE_TOKEN_PRODUCT_ID
		ALL_SPOTS_FOR_INSURANCE = []uint32{
			ARB_USDC,
			ETH_USDT,
			ARB_USDT,
			OP_USDC,
			OP_USDT,
			BASE_USDC,
			BASE_USDT,
			MANTLE_USDC,
			MANTLE_USDT,
			BERACHAIN_USDC,
			BERACHAIN_USDT,
			MODE_USDC,
			MODE_USDT,
			KROMA_USDC,
			KROMA_USDT,
			MONADTESTNET_USDC,
		}
		ALL_NON_PREDICTION_PERPS = []uint32{
			ETH_MARKET,
			BTC_MARKET,
			SOL_MARKET,
			PEPE_MARKET,
			DOGE_MARKET,
			ARB_MARKET,
			LINK_MARKET,
			XRP_MARKET,
			NEAR_MARKET,
			EIGEN_MARKET,
			TON_MARKET,
			GOAT_MARKET,
			SPX_MARKET,
			SCR_MARKET,
			WIF_MARKET,
			POPCAT_MARKET,
			NEIRO_MARKET,
			kPEPE_MARKET,
			kSHIB_MARKET,
			FARTCOIN_MARKET,
			MOODENG_MARKET,
			GIGA_MARKET,
			kBONK_MARKET,
			BRETT_MARKET,
			GRASS_MARKET,
			TIA_MARKET,
			SEI_MARKET,
			SUI_MARKET,
			ZRO_MARKET,
			TAIKO_MARKET,
			VIRTUAL_MARKET,
			BOME_MARKET,
			APE_MARKET,
			PONKE_MARKET,
			NPC_MARKET,
			SWELL_MARKET,
			JPY_MARKET,
			ALGO_MARKET,
			XLM_MARKET,
			HBAR_MARKET,
			EUR_MARKET,
			XAUT_MARKET,
			MOVE_MARKET,
			LOGX_MARKET,
			OFFICIAL_TRUMP_MARKET,
			WAL_MARKET,
			FLUID_MARKET,
			PROMPT_MARKET,
			NXPC_MARKET,
			TSLA_MARKET,
			NVDA_MARKET,
			AAPL_MARKET,
			QQQ_MARKET,
			BTC_OSTRICH_MARKET,
			ETH_OSTRICH_MARKET,
			SOL_OSTRICH_MARKET,
			XAUT_OSTRICH_MARKET,
			JPY_OSTRICH_MARKET,
			EUR_OSTRICH_MARKET,
			GBP_OSTRICH_MARKET,
			COIN_OSTRICH_MARKET,
			GOOGL_OSTRICH_MARKET,
			MSFT_OSTRICH_MARKET,
			AMZN_OSTRICH_MARKET,
			META_OSTRICH_MARKET,
			MSTR_OSTRICH_MARKET,
			DEFT_OSTRICH_MARKET,
			PLTR_OSTRICH_MARKET,
			HOOD_OSTRICH_MARKET,
			PYPL_OSTRICH_MARKET,
			GME_OSTRICH_MARKET,
			RIOT_OSTRICH_MARKET,
			COKE_OSTRICH_MARKET,
			CRCL_OSTRICH_MARKET,
			RDDT_OSTRICH_MARKET,
			USOIL_OSTRICH_MARKET,
			COPPER_OSTRICH_MARKET,
			NGAS_OSTRICH_MARKET,
			Silver_OSTRICH_MARKET,
			HIMS_OSTRICH_MARKET,
			SPY_OSTRICH_MARKET,
			NA_OSTRICH_MARKET,
			HUT_OSTRICH_MARKET,
			BRK_B_OSTRICH_MARKET,
			TCEHY_OSTRICH_MARKET,
			BIDU_OSTRICH_MARKET,
			BABA_OSTRICH_MARKET,
			NKE_OSTRICH_MARKET,
			SOFI_OSTRICH_MARKET,
			PUMP_MARKET,
			NIO_OSTRICH_MARKET,
			JD_OSTRICH_MARKET,
			TRY_OSTRICH_MARKET,
			AUD_OSTRICH_MARKET,
			NZD_OSTRICH_MARKET,
			CHF_OSTRICH_MARKET,
			YHC_OSTRICH_MARKET,
			TRON_OSTRICH_MARKET,
			SONY_OSTRICH_MARKET,
			MTPLF_OSTRICH_MARKET,
			MUFG_OSTRICH_MARKET,
			TM_OSTRICH_MARKET,
			HMC_OSTRICH_MARKET,
			AZN_OSTRICH_MARKET,
			HSBC_OSTRICH_MARKET,
			SHEL_OSTRICH_MARKET,
			BCS_OSTRICH_MARKET,
			RYCEY_OSTRICH_MARKET,
			RICH_OSTRICH_MARKET,
			XYZ100_MARKET,
		}
		ALL_PREDICTION_PERPS = []uint32{
			TRUMP_MARKET,
			HARRIS_MARKET,
		}
		ALL_MEME_PERPS = []uint32{
			DOGE_MARKET,
			GOAT_MARKET,
			WIF_MARKET,
			POPCAT_MARKET,
			NEIRO_MARKET,
			kPEPE_MARKET,
			kSHIB_MARKET,
			FARTCOIN_MARKET,
			MOODENG_MARKET,
			GIGA_MARKET,
			kBONK_MARKET,
			BRETT_MARKET,
			BOME_MARKET,
			PONKE_MARKET,
			NPC_MARKET,
			OFFICIAL_TRUMP_MARKET,
		}

		ALL_PERPS_ON_CONTRACT = []uint32{
			ETH_MARKET,
			BTC_MARKET,
			SOL_MARKET,
			PEPE_MARKET,
			DOGE_MARKET,
			TRUMP_MARKET,
			HARRIS_MARKET,
			ARB_MARKET,
			LINK_MARKET,
			XRP_MARKET,
			NEAR_MARKET,
			EIGEN_MARKET,
			TON_MARKET,
			GOAT_MARKET,
			SPX_MARKET,
			SCR_MARKET,
			WIF_MARKET,
			POPCAT_MARKET,
			NEIRO_MARKET,
			kPEPE_MARKET,
			kSHIB_MARKET,
			FARTCOIN_MARKET,
			MOODENG_MARKET,
			GIGA_MARKET,
			kBONK_MARKET,
			BRETT_MARKET,
			GRASS_MARKET,
			TIA_MARKET,
			SEI_MARKET,
			SUI_MARKET,
			ZRO_MARKET,
			TAIKO_MARKET,
			VIRTUAL_MARKET,
			BOME_MARKET,
			APE_MARKET,
			PONKE_MARKET,
			NPC_MARKET,
			SWELL_MARKET,
			JPY_MARKET,
			ALGO_MARKET,
			XLM_MARKET,
			HBAR_MARKET,
			EUR_MARKET,
			XAUT_MARKET,
			MOVE_MARKET,
			LOGX_MARKET,
			OFFICIAL_TRUMP_MARKET,
			WAL_MARKET,
			FLUID_MARKET,
			PROMPT_MARKET,
			NXPC_MARKET,
			TSLA_MARKET,
			NVDA_MARKET,
			AAPL_MARKET,
			QQQ_MARKET,
			BTC_OSTRICH_MARKET,
			ETH_OSTRICH_MARKET,
			SOL_OSTRICH_MARKET,
			XAUT_OSTRICH_MARKET,
			JPY_OSTRICH_MARKET,
			EUR_OSTRICH_MARKET,
			GBP_OSTRICH_MARKET,
			COIN_OSTRICH_MARKET,
			GOOGL_OSTRICH_MARKET,
			MSFT_OSTRICH_MARKET,
			AMZN_OSTRICH_MARKET,
			META_OSTRICH_MARKET,
			MSTR_OSTRICH_MARKET,
			DEFT_OSTRICH_MARKET,
			PLTR_OSTRICH_MARKET,
			HOOD_OSTRICH_MARKET,
			PYPL_OSTRICH_MARKET,
			GME_OSTRICH_MARKET,
			RIOT_OSTRICH_MARKET,
			COKE_OSTRICH_MARKET,
			CRCL_OSTRICH_MARKET,
			RDDT_OSTRICH_MARKET,
			USOIL_OSTRICH_MARKET,
			COPPER_OSTRICH_MARKET,
			NGAS_OSTRICH_MARKET,
			Silver_OSTRICH_MARKET,
			HIMS_OSTRICH_MARKET,
			SPY_OSTRICH_MARKET,
			NA_OSTRICH_MARKET,
			HUT_OSTRICH_MARKET,
			BRK_B_OSTRICH_MARKET,
			TCEHY_OSTRICH_MARKET,
			BIDU_OSTRICH_MARKET,
			BABA_OSTRICH_MARKET,
			NKE_OSTRICH_MARKET,
			SOFI_OSTRICH_MARKET,
			PUMP_MARKET,
			NIO_OSTRICH_MARKET,
			JD_OSTRICH_MARKET,
			TRY_OSTRICH_MARKET,
			AUD_OSTRICH_MARKET,
			NZD_OSTRICH_MARKET,
			CHF_OSTRICH_MARKET,
			YHC_OSTRICH_MARKET,
			TRON_OSTRICH_MARKET,
			SONY_OSTRICH_MARKET,
			MTPLF_OSTRICH_MARKET,
			MUFG_OSTRICH_MARKET,
			TM_OSTRICH_MARKET,
			HMC_OSTRICH_MARKET,
			AZN_OSTRICH_MARKET,
			HSBC_OSTRICH_MARKET,
			SHEL_OSTRICH_MARKET,
			BCS_OSTRICH_MARKET,
			RYCEY_OSTRICH_MARKET,
			RICH_OSTRICH_MARKET,
			XYZ100_MARKET,
		}
		ALL_COLLATERAL_TOKEN_SYMBOLS = []string{
			"USDC",
			"USDT",
		}
		ARBITRUM = 421614
		ETHEREUM = 11155111
		OPTIMISM = 11155420
		BASE = 84532
		MANTLE = 5003
		MODE = 919
		BERACHAIN = 80084
		MONADTESTNET = 10143
		ProductChainMapping = map[uint32]int64{
			0:  ARBITRUM,
			10: ARBITRUM,
			6:  ARBITRUM,
			8:  ETHEREUM,
			4:  ETHEREUM,
			14: OPTIMISM,
			12: OPTIMISM,
			18: BASE,
			16: BASE,
			22: MANTLE,
			20: MANTLE,
			30: MODE,
			28: MODE,
			26: BERACHAIN,
			24: BERACHAIN,
			36: MONADTESTNET,
		}
		SUPPORTED_LOGX_CHAINS = map[int64]bool{
			ARBITRUM_CHAIN_ID: true,
		}
		PRODUCT_ID_TO_SOURCE_CHAIN_INFO = map[uint32]SourceChainInfo{
			0:  {"0x9456fA8e1aE8381951D7c20dA0F80C3D983dac4d", "https://arb-sepolia.g.alchemy.com/v2/IVlEQMOELGwatIkr1a-_KcFi3YI2Xiuq"},
			4:  {"0xfFAEF09B3cd11D9b20d1a19bECca54EEC2884766", "https://eth-sepolia.g.alchemy.com/v2/rSRWdoQWjy-Dd0fj1H94PAJ3lVbsZ3QC"},
			6:  {"0x9456fA8e1aE8381951D7c20dA0F80C3D983dac4d", "https://arb-sepolia.g.alchemy.com/v2/IVlEQMOELGwatIkr1a-_KcFi3YI2Xiuq"},
			8:  {"0xfFAEF09B3cd11D9b20d1a19bECca54EEC2884766", "https://eth-sepolia.g.alchemy.com/v2/rSRWdoQWjy-Dd0fj1H94PAJ3lVbsZ3QC"},
			10: {"0x9456fA8e1aE8381951D7c20dA0F80C3D983dac4d", "https://arb-sepolia.g.alchemy.com/v2/IVlEQMOELGwatIkr1a-_KcFi3YI2Xiuq"},
			28: {"0x8873c061aACE95e52Abf6Ec59BCa90B6f1F16be9", "https://sepolia.mode.network"},
			30: {"0x8873c061aACE95e52Abf6Ec59BCa90B6f1F16be9", "https://sepolia.mode.network"},
			36: {"0xFAeA787E61af0323A2F7D6c01ac27C9322198137", "https://monad-testnet.g.alchemy.com/v2/sLRzTU4Z-kpvxGx8d7_UXb-6qP_uqQIh"},
		}

		// IF not provided it will take the default value
		WITHDRAWAL_FEE_MAP = map[uint32]string{
			0:  "25",
			4:  "0.5",
			6:  "0.5",
			16: "0.5",
			46: "0.5",
		}

		OPTIONS_INTERVALS = map[uint32]bool{
			1:  true,
			2:  true,
			5:  true,
			30: true,
		}

		SYMBOL_TO_OSTRICH_SYMBOL = map[string]string{
			"BTC":    "BTC_OSTRICH",
			"ETH":    "ETH_OSTRICH",
			"SOL":    "SOL_OSTRICH",
			"XAUT":   "XAUT_OSTRICH",
			"JPY":    "JPY_OSTRICH",
			"EUR":    "EUR_OSTRICH",
			"GBP":    "GBP_OSTRICH",
			"COIN":   "COIN_OSTRICH",
			"GOOGL":  "GOOGL_OSTRICH",
			"MSFT":   "MSFT_OSTRICH",
			"AMZN":   "AMZN_OSTRICH",
			"META":   "META_OSTRICH",
			"MSTR":   "MSTR_OSTRICH",
			"DEFT":   "DEFT_OSTRICH",
			"PLTR":   "PLTR_OSTRICH",
			"HOOD":   "HOOD_OSTRICH",
			"PYPL":   "PYPL_OSTRICH",
			"GME":    "GME_OSTRICH",
			"RIOT":   "RIOT_OSTRICH",
			"COKE":   "COKE_OSTRICH",
			"CRCL":   "CRCL_OSTRICH",
			"RDDT":   "RDDT_OSTRICH",
			"USOIL":  "USOIL_OSTRICH",
			"COPPER": "COPPER_OSTRICH",
			"NGAS":   "NGAS_OSTRICH",
			"Silver": "Silver_OSTRICH",
			"HIMS":   "HIMS_OSTRICH",
			"SPY":    "SPY_OSTRICH",
			"NA":     "NA_OSTRICH",
			"HUT":    "HUT_OSTRICH",
			"BRK.B":  "BRK_B_OSTRICH",
			"TCEHY":  "TCEHY_OSTRICH",
			"BIDU":   "BIDU_OSTRICH",
			"BABA":   "BABA_OSTRICH",
			"NKE":    "NKE_OSTRICH",
			"SOFI":   "SOFI_OSTRICH",
			"NIO":    "NIO_OSTRICH",
			"JD":     "JD_OSTRICH",
			"TRY":    "TRY_OSTRICH",
			"AUD":    "AUD_OSTRICH",
			"NZD":    "NZD_OSTRICH",
			"CHF":    "CHF_OSTRICH",
			"YHC":    "YHC_OSTRICH",
			"TRON":   "TRON_OSTRICH",
			"SONY":   "SONY_OSTRICH",
			"MTPLF":  "MTPLF_OSTRICH",
			"MUFG":   "MUFG_OSTRICH",
			"TM":     "TM_OSTRICH",
			"HMC":    "HMC_OSTRICH",
			"AZN":    "AZN_OSTRICH",
			"HSBC":   "HSBC_OSTRICH",
			"SHEL":   "SHEL_OSTRICH",
			"BCS":    "BCS_OSTRICH",
			"RYCEY":  "RYCEY_OSTRICH",
			"RICH":   "RICH_OSTRICH",
		}
		PRODUCT_ID_TO_MAILBOX_ADDRESS = make(map[uint32]string)
		for productId := range PRODUCT_ID_TO_SOURCE_CHAIN_INFO {
			PRODUCT_ID_TO_MAILBOX_ADDRESS[productId] = LOGX_MAILBOX_ADDRESS
		}
		PRODUCT_ID_TO_MAILBOX_ADDRESS[74] = OSTRICH_MAILBOX_ADDRESS

	case MAINNET:
		RPC_URL = "https://vzjuxmhfn70kgnlds27h.alt.technology"
		ENDPOINT_CONTRACT_ADDRESS = "0xBC87C2397601391E66adeC581786dF3F8eeE6124"
		OFFCHAIN_CONTRACT_ADDRESS = "0x1182cEa3449264CbC40A3f2A411AFFf941F84455"
		STAKER_CONTRACT_ADDRESS = "0x4CfB0D13f9dcCc9Dac6195FE9644878e8a28f3ce"
		BRIDGE_OUT_CONTRACT = "0x58Ce6D64983C53B5Fcf1141d3A54C0376022dac5"
		CLEARING_HOUSE = "0x22b6E26F8204e67c908E54801b13e14cb1BD9872"
		PERP_CONTRACT_ADDRESS = "0x15F1895e500186746B77F46749121c409055ecD1"
		SPOT_CONTRACT_ADDRESS = "0x1a9cFF67a0370000F6feF3F40f15422331923db7"
		LOGX_MAILBOX_ADDRESS = "0xfa5701B7D8fa4A8bb0BAfFB304F676b6f8101cFf"
		OSTRICH_MAILBOX_ADDRESS = "0xf539b5B8a9C496067D27617dCf3Fe3fe562dDc08"
		ARB_USDC = 4
		OSTRICH_USDC_UPDATED = 74
		LOGX_CHAIN_ID = 936369
		ARBITRUM_CHAIN_ID = 42161
		ETHEREUM_CHAIN_ID = 1
		LIQUIDATION_FRACTION_STR = map[uint32]string{
			GOAT_MARKET:           "0.05", // 5%
			SPX_MARKET:            "0.05", // 5%
			WIF_MARKET:            "0.05", // 5%
			POPCAT_MARKET:         "0.05", // 5%
			NEIRO_MARKET:          "0.05", // 5%
			kPEPE_MARKET:          "0.05", // 5%
			kSHIB_MARKET:          "0.05", // 5%
			FARTCOIN_MARKET:       "0.05", // 5%
			MOODENG_MARKET:        "0.05", // 5%
			GIGA_MARKET:           "0.05", // 5%
			kBONK_MARKET:          "0.05", // 5%
			BRETT_MARKET:          "0.05", // 5%
			GRASS_MARKET:          "0.05", // 5%
			BOME_MARKET:           "0.05", // 5%
			APE_MARKET:            "0.05", // 5%
			PONKE_MARKET:          "0.05", // 5%
			NPC_MARKET:            "0.05", // 5%
			OFFICIAL_TRUMP_MARKET: "0.10", // 10%
			WAL_MARKET:            "0.03", // 3%
			FLUID_MARKET:          "0.04", // 4%
			PROMPT_MARKET:         "0.04", // 4%
			NXPC_MARKET:           "0.04", // 4%
			TSLA_MARKET:           "0.02",
			NVDA_MARKET:           "0.008",
			AAPL_MARKET:           "0.008",
			QQQ_MARKET:            "0.02",
			BTC_OSTRICH_MARKET:    "0.01", // 1%
			ETH_OSTRICH_MARKET:    "0.01", // 1%
			SOL_OSTRICH_MARKET:    "0.01", // 1%
			XAUT_OSTRICH_MARKET:   "0.02", // 2%
			JPY_OSTRICH_MARKET:    "0.02", // 2%
			EUR_OSTRICH_MARKET:    "0.02", // 2%
			GBP_OSTRICH_MARKET:    "0.02", // 2%
			COIN_OSTRICH_MARKET:   "0.02", // 2%
			GOOGL_OSTRICH_MARKET:  "0.02", // 2%
			MSFT_OSTRICH_MARKET:   "0.02", // 2%
			AMZN_OSTRICH_MARKET:   "0.02", // 2%
			META_OSTRICH_MARKET:   "0.02", // 2%
			MSTR_OSTRICH_MARKET:   "0.02", // 2%
			DEFT_OSTRICH_MARKET:   "0.02", // 2%
			PLTR_OSTRICH_MARKET:   "0.04", // 4%
			HOOD_OSTRICH_MARKET:   "0.02", // 2%
			PYPL_OSTRICH_MARKET:   "0.02", // 2%
			GME_OSTRICH_MARKET:    "0.02", // 2%
			RIOT_OSTRICH_MARKET:   "0.02", // 2%
			COKE_OSTRICH_MARKET:   "0.02", // 2%
			CRCL_OSTRICH_MARKET:   "0.02", // 2%
			RDDT_OSTRICH_MARKET:   "0.02", // 2%
			USOIL_OSTRICH_MARKET:  "0.02", // 2%
			COPPER_OSTRICH_MARKET: "0.02", // 2%
			NGAS_OSTRICH_MARKET:   "0.02", // 2%
			Silver_OSTRICH_MARKET: "0.02", // 2%
			HIMS_OSTRICH_MARKET:   "0.02", // 2%
			SPY_OSTRICH_MARKET:    "0.02", // 2%
			NA_OSTRICH_MARKET:     "0.02", // 2%
			HUT_OSTRICH_MARKET:    "0.02", // 2%
			BRK_B_OSTRICH_MARKET:  "0.02", // 2%
			TCEHY_OSTRICH_MARKET:  "0.02", // 2%
			BIDU_OSTRICH_MARKET:   "0.02", // 2%
			BABA_OSTRICH_MARKET:   "0.02", // 2%
			NKE_OSTRICH_MARKET:    "0.02", // 2%
			SOFI_OSTRICH_MARKET:   "0.02", // 2%
			PUMP_MARKET:           "0.05", // 5%
			XYZ100_MARKET:         "0.05", // 5%
			NIO_OSTRICH_MARKET:    "0.02", // 2%
			JD_OSTRICH_MARKET:     "0.02", // 2%
			TRY_OSTRICH_MARKET:    "0.02", // 2%
			AUD_OSTRICH_MARKET:    "0.02", // 2%
			NZD_OSTRICH_MARKET:    "0.02", // 2%
			CHF_OSTRICH_MARKET:    "0.02", // 2%
			YHC_OSTRICH_MARKET:    "0.05", // 2%
			TRON_OSTRICH_MARKET:   "0.05", // 5%
			SONY_OSTRICH_MARKET:   "0.02", // 2%
			MTPLF_OSTRICH_MARKET:  "0.02", // 2%
			MUFG_OSTRICH_MARKET:   "0.02", // 2%
			TM_OSTRICH_MARKET:     "0.02", // 2%
			HMC_OSTRICH_MARKET:    "0.02", // 2%
			AZN_OSTRICH_MARKET:    "0.02", // 2%
			HSBC_OSTRICH_MARKET:   "0.02", // 2%
			SHEL_OSTRICH_MARKET:   "0.02", // 2%
			BCS_OSTRICH_MARKET:    "0.02", // 2%
			RYCEY_OSTRICH_MARKET:  "0.02", // 2%
			RICH_OSTRICH_MARKET:   "0.02", // 2%
		}
		SOURCE_HYPERC_ADDRESS_MAP = map[uint]string{
			4:  "0x4882520D47491561F51ea96aBC0397776Efc6cFd",
			74: "0xD7cF7165c4F0b7B34E03D04eD3cB730850d508fB",
		}

		PRODUCT_ID_SYMBOL_TO_MAP = map[uint32]string{
			4:     "USDC",
			74:    "USDC",
			1:     "ETH",
			3:     "BTC",
			5:     "SOL",
			7:     "PEPE",
			9:     "DOGE",
			11:    "XTRUMP",
			13:    "HARRIS",
			15:    "ARB",
			17:    "LINK",
			19:    "XRP",
			21:    "NEAR",
			23:    "EIGEN",
			25:    "TON",
			27:    "GOAT",
			29:    "SPX",
			31:    "SCR",
			33:    "WIF",
			35:    "POPCAT",
			37:    "NEIRO",
			39:    "kPEPE",
			41:    "kSHIB",
			43:    "FARTCOIN",
			45:    "MOODENG",
			47:    "GIGA",
			49:    "kBONK",
			51:    "BRETT",
			53:    "GRASS",
			55:    "TIA",
			57:    "SEI",
			59:    "SUI",
			61:    "ZRO",
			63:    "TAIKO",
			65:    "VIRTUAL",
			67:    "BOME",
			69:    "APE",
			71:    "PONKE",
			73:    "NPC",
			75:    "SWELL",
			77:    "JPY",
			79:    "ALGO",
			81:    "XLM",
			83:    "HBAR",
			85:    "EUR",
			87:    "XAUT",
			89:    "MOVE",
			91:    "LOGX",
			93:    "TRUMP",
			95:    "WAL",
			97:    "FLUID",
			99:    "PROMPT",
			101:   "NXPC",
			103:   "TSLA",
			105:   "NVDA",
			107:   "AAPL",
			109:   "QQQ",
			111:   "BTC_OSTRICH",
			113:   "ETH_OSTRICH",
			115:   "SOL_OSTRICH",
			117:   "XAUT_OSTRICH",
			119:   "JPY_OSTRICH",
			121:   "EUR_OSTRICH",
			123:   "GBP_OSTRICH",
			125:   "COIN_OSTRICH",
			127:   "GOOGL_OSTRICH",
			129:   "MSFT_OSTRICH",
			131:   "AMZN_OSTRICH",
			133:   "META_OSTRICH",
			135:   "MSTR_OSTRICH",
			137:   "DEFT_OSTRICH",
			139:   "PLTR_OSTRICH",
			141:   "HOOD_OSTRICH",
			143:   "PYPL_OSTRICH",
			145:   "GME_OSTRICH",
			147:   "RIOT_OSTRICH",
			149:   "COKE_OSTRICH",
			151:   "CRCL_OSTRICH",
			153:   "RDDT_OSTRICH",
			155:   "USOIL_OSTRICH",
			157:   "COPPER_OSTRICH",
			159:   "NGAS_OSTRICH",
			161:   "Silver_OSTRICH",
			163:   "HIMS_OSTRICH",
			165:   "SPY_OSTRICH",
			167:   "NA_OSTRICH",
			169:   "HUT_OSTRICH",
			171:   "BRK_B_OSTRICH",
			173:   "TCEHY_OSTRICH",
			175:   "BIDU_OSTRICH",
			177:   "BABA_OSTRICH",
			179:   "NKE_OSTRICH",
			181:   "SOFI_OSTRICH",
			183:   "PUMP",
			185:   "NIO_OSTRICH",
			187:   "JD_OSTRICH",
			189:   "TRY_OSTRICH",
			191:   "AUD_OSTRICH",
			193:   "NZD_OSTRICH",
			195:   "CHF_OSTRICH",
			197:   "YHC_OSTRICH",
			199:   "TRON_OSTRICH",
			201:   "SONY_OSTRICH",
			203:   "MTPLF_OSTRICH",
			205:   "MUFG_OSTRICH",
			207:   "TM_OSTRICH",
			209:   "HMC_OSTRICH",
			211:   "AZN_OSTRICH",
			213:   "HSBC_OSTRICH",
			215:   "SHEL_OSTRICH",
			217:   "BCS_OSTRICH",
			219:   "RYCEY_OSTRICH",
			221:   "RICH_OSTRICH",
			223:   "XYZ100",
			20001: "BERA",
			20002: "LOGX",
			20003: "ETH",
			20004: "BTC",
			20005: "IP",
			20006: "KAITO",
			20007: "RED",
			20008: "ELX",
			20009: "MUBARAK",
			20010: "K",
			20011: "INIT",
			10001: "ETH",
			10003: "BTC",
			10005: "SOL",
			10007: "HYPE",
		}
		SPOT_TO_PRE_MARKET_MAP = map[uint32]uint32{
			20001: 1001,
			20005: 1003,
			20006: 1008,
			20007: 1004,
			20010: 1012,
			20011: 1014,
		}
		PRE_MARKET_TO_SPOT_FACTOR_MAP = map[uint32]uint32{
			1001: 50,
			1003: 100,
			1008: 100,
			1004: 100,
			1012: 1,
			1014: 100,
		}
		PRODUCT_MARKET_WEIGHTS = map[uint32]int{
			ARB_USDC:             1, //4
			OSTRICH_USDC_UPDATED: 1, //74
		}
		ALL_SPOTS_IN_ORDER = []uint32{
			ARB_USDC,
			OSTRICH_USDC_UPDATED,
		}
		ALL_SPOTS_ON_CONTRACT = []uint32{
			ARB_USDC,
			OSTRICH_USDC_UPDATED,
		}
		ALL_COLLATERAL_SPOTS = []uint32{
			ARB_USDC,
			OSTRICH_USDC_UPDATED,
		}
		// Do not include LogX, XLogX and QUOTE_TOKEN_PRODUCT_ID
		ALL_SPOTS_FOR_INSURANCE = []uint32{
			OSTRICH_USDC_UPDATED,
		}
		ALL_NON_PREDICTION_PERPS = []uint32{
			ETH_MARKET,
			BTC_MARKET,
			SOL_MARKET,
			PEPE_MARKET,
			DOGE_MARKET,
			ARB_MARKET,
			LINK_MARKET,
			XRP_MARKET,
			NEAR_MARKET,
			EIGEN_MARKET,
			TON_MARKET,
			GOAT_MARKET,
			SPX_MARKET,
			SCR_MARKET,
			WIF_MARKET,
			POPCAT_MARKET,
			NEIRO_MARKET,
			kPEPE_MARKET,
			kSHIB_MARKET,
			FARTCOIN_MARKET,
			MOODENG_MARKET,
			GIGA_MARKET,
			kBONK_MARKET,
			BRETT_MARKET,
			GRASS_MARKET,
			TIA_MARKET,
			SEI_MARKET,
			SUI_MARKET,
			ZRO_MARKET,
			TAIKO_MARKET,
			VIRTUAL_MARKET,
			BOME_MARKET,
			APE_MARKET,
			PONKE_MARKET,
			NPC_MARKET,
			SWELL_MARKET,
			JPY_MARKET,
			ALGO_MARKET,
			XLM_MARKET,
			HBAR_MARKET,
			EUR_MARKET,
			XAUT_MARKET,
			MOVE_MARKET,
			LOGX_MARKET,
			OFFICIAL_TRUMP_MARKET,
			WAL_MARKET,
			FLUID_MARKET,
			PROMPT_MARKET,
			NXPC_MARKET,
			TSLA_MARKET,
			NVDA_MARKET,
			AAPL_MARKET,
			QQQ_MARKET,
			BTC_OSTRICH_MARKET,
			ETH_OSTRICH_MARKET,
			SOL_OSTRICH_MARKET,
			XAUT_OSTRICH_MARKET,
			JPY_OSTRICH_MARKET,
			EUR_OSTRICH_MARKET,
			GBP_OSTRICH_MARKET,
			COIN_OSTRICH_MARKET,
			GOOGL_OSTRICH_MARKET,
			MSFT_OSTRICH_MARKET,
			AMZN_OSTRICH_MARKET,
			META_OSTRICH_MARKET,
			MSTR_OSTRICH_MARKET,
			DEFT_OSTRICH_MARKET,
			PLTR_OSTRICH_MARKET,
			HOOD_OSTRICH_MARKET,
			PYPL_OSTRICH_MARKET,
			GME_OSTRICH_MARKET,
			RIOT_OSTRICH_MARKET,
			COKE_OSTRICH_MARKET,
			CRCL_OSTRICH_MARKET,
			RDDT_OSTRICH_MARKET,
			USOIL_OSTRICH_MARKET,
			COPPER_OSTRICH_MARKET,
			NGAS_OSTRICH_MARKET,
			Silver_OSTRICH_MARKET,
			HIMS_OSTRICH_MARKET,
			SPY_OSTRICH_MARKET,
			NA_OSTRICH_MARKET,
			HUT_OSTRICH_MARKET,
			BRK_B_OSTRICH_MARKET,
			TCEHY_OSTRICH_MARKET,
			BIDU_OSTRICH_MARKET,
			BABA_OSTRICH_MARKET,
			NKE_OSTRICH_MARKET,
			SOFI_OSTRICH_MARKET,
			PUMP_MARKET,
			NIO_OSTRICH_MARKET,
			JD_OSTRICH_MARKET,
			TRY_OSTRICH_MARKET,
			AUD_OSTRICH_MARKET,
			NZD_OSTRICH_MARKET,
			CHF_OSTRICH_MARKET,
			YHC_OSTRICH_MARKET,
			TRON_OSTRICH_MARKET,
			SONY_OSTRICH_MARKET,
			MTPLF_OSTRICH_MARKET,
			MUFG_OSTRICH_MARKET,
			TM_OSTRICH_MARKET,
			HMC_OSTRICH_MARKET,
			AZN_OSTRICH_MARKET,
			HSBC_OSTRICH_MARKET,
			SHEL_OSTRICH_MARKET,
			BCS_OSTRICH_MARKET,
			RYCEY_OSTRICH_MARKET,
			RICH_OSTRICH_MARKET,
			XYZ100_MARKET,
		}
		ALL_PREDICTION_PERPS = []uint32{
			TRUMP_MARKET,
			HARRIS_MARKET,
		}
		ALL_MEME_PERPS = []uint32{
			DOGE_MARKET,
			GOAT_MARKET,
			WIF_MARKET,
			POPCAT_MARKET,
			NEIRO_MARKET,
			kPEPE_MARKET,
			kSHIB_MARKET,
			FARTCOIN_MARKET,
			MOODENG_MARKET,
			GIGA_MARKET,
			kBONK_MARKET,
			BRETT_MARKET,
			BOME_MARKET,
			PONKE_MARKET,
			NPC_MARKET,
			OFFICIAL_TRUMP_MARKET,
		}
		ALL_PERPS_ON_CONTRACT = []uint32{
			ETH_MARKET,
			BTC_MARKET,
			SOL_MARKET,
			PEPE_MARKET,
			DOGE_MARKET,
			TRUMP_MARKET,
			HARRIS_MARKET,
			ARB_MARKET,
			LINK_MARKET,
			XRP_MARKET,
			NEAR_MARKET,
			EIGEN_MARKET,
			TON_MARKET,
			GOAT_MARKET,
			SPX_MARKET,
			SCR_MARKET,
			WIF_MARKET,
			POPCAT_MARKET,
			NEIRO_MARKET,
			kPEPE_MARKET,
			kSHIB_MARKET,
			FARTCOIN_MARKET,
			MOODENG_MARKET,
			GIGA_MARKET,
			kBONK_MARKET,
			BRETT_MARKET,
			GRASS_MARKET,
			TIA_MARKET,
			SEI_MARKET,
			SUI_MARKET,
			ZRO_MARKET,
			TAIKO_MARKET,
			VIRTUAL_MARKET,
			BOME_MARKET,
			APE_MARKET,
			PONKE_MARKET,
			NPC_MARKET,
			SWELL_MARKET,
			JPY_MARKET,
			ALGO_MARKET,
			XLM_MARKET,
			HBAR_MARKET,
			EUR_MARKET,
			XAUT_MARKET,
			MOVE_MARKET,
			LOGX_MARKET,
			OFFICIAL_TRUMP_MARKET,
			WAL_MARKET,
			FLUID_MARKET,
			PROMPT_MARKET,
			NXPC_MARKET,
			TSLA_MARKET,
			NVDA_MARKET,
			AAPL_MARKET,
			QQQ_MARKET,
			BTC_OSTRICH_MARKET,
			ETH_OSTRICH_MARKET,
			SOL_OSTRICH_MARKET,
			XAUT_OSTRICH_MARKET,
			JPY_OSTRICH_MARKET,
			EUR_OSTRICH_MARKET,
			GBP_OSTRICH_MARKET,
			COIN_OSTRICH_MARKET,
			GOOGL_OSTRICH_MARKET,
			MSFT_OSTRICH_MARKET,
			AMZN_OSTRICH_MARKET,
			META_OSTRICH_MARKET,
			MSTR_OSTRICH_MARKET,
			DEFT_OSTRICH_MARKET,
			PLTR_OSTRICH_MARKET,
			HOOD_OSTRICH_MARKET,
			PYPL_OSTRICH_MARKET,
			GME_OSTRICH_MARKET,
			RIOT_OSTRICH_MARKET,
			COKE_OSTRICH_MARKET,
			CRCL_OSTRICH_MARKET,
			RDDT_OSTRICH_MARKET,
			USOIL_OSTRICH_MARKET,
			COPPER_OSTRICH_MARKET,
			NGAS_OSTRICH_MARKET,
			Silver_OSTRICH_MARKET,
			HIMS_OSTRICH_MARKET,
			SPY_OSTRICH_MARKET,
			NA_OSTRICH_MARKET,
			HUT_OSTRICH_MARKET,
			BRK_B_OSTRICH_MARKET,
			TCEHY_OSTRICH_MARKET,
			BIDU_OSTRICH_MARKET,
			BABA_OSTRICH_MARKET,
			NKE_OSTRICH_MARKET,
			SOFI_OSTRICH_MARKET,
			PUMP_MARKET,
			NIO_OSTRICH_MARKET,
			JD_OSTRICH_MARKET,
			TRY_OSTRICH_MARKET,
			AUD_OSTRICH_MARKET,
			NZD_OSTRICH_MARKET,
			CHF_OSTRICH_MARKET,
			YHC_OSTRICH_MARKET,
			TRON_OSTRICH_MARKET,
			SONY_OSTRICH_MARKET,
			MTPLF_OSTRICH_MARKET,
			MUFG_OSTRICH_MARKET,
			TM_OSTRICH_MARKET,
			HMC_OSTRICH_MARKET,
			AZN_OSTRICH_MARKET,
			HSBC_OSTRICH_MARKET,
			SHEL_OSTRICH_MARKET,
			BCS_OSTRICH_MARKET,
			RYCEY_OSTRICH_MARKET,
			RICH_OSTRICH_MARKET,
			XYZ100_MARKET,
		}
		ALL_COLLATERAL_TOKEN_SYMBOLS = []string{
			"USDC",
			"USDT",
		}
		ARBITRUM = 42161
		MANTLE = 5000
		MODE = 34443
		BASE = 8453
		BOB = 60808
		SEI = 1329
		SCROLL = 534352
		KROMA = 255
		TAIKO = 167000
		OPTIMISM = 10
		LINEA = 59144
		RARI = 1380012617
		MINT = 185
		POLYGON = 137
		ABSTRACT = 2741
		APECHAIN = 33139
		SONIC = 146
		MANTA = 169
		BLAST = 81457
		ProductChainMapping = map[uint32]int64{
			4:  ARBITRUM,
			74: ARBITRUM,
		}

		SUPPORTED_LOGX_CHAINS = map[int64]bool{
			ARBITRUM_CHAIN_ID: true,
		}
		PRODUCT_ID_TO_SOURCE_CHAIN_INFO = map[uint32]SourceChainInfo{
			4:  {"0x873c672fd3B440DB722a97586b483F0459751254", "https://arb1.arbitrum.io/rpc"},
			74: {"0x584D9f39DA7dAB0589B72a892C0C4814beaCb3a8", "https://arb1.arbitrum.io/rpc"},
		}
		// IF not provided it will take the default value

		WITHDRAWAL_FEE_MAP = map[uint32]string{
			4:  "0.5",
			74: "1",
		}
		RELAYER_CHAINS = []Chain{
			{
				Name:      "Arbitrum",
				Rpc:       "https://1rpc.io/arb",
				NativeId:  "ETH",
				Threshold: 100,
			},
			{
				Name:      "Mantle",
				Rpc:       "https://rpc.mantle.xyz",
				NativeId:  "MNT",
				Threshold: 100,
			},
			{
				Name:      "Mode",
				Rpc:       "https://mainnet.mode.network",
				NativeId:  "ETH",
				Threshold: 100,
			},
			{
				Name:      "Base",
				Rpc:       "https://base.llamarpc.com",
				NativeId:  "ETH",
				Threshold: 100,
			},
			{
				Name:      "BoB",
				Rpc:       "https://rpc.gobob.xyz",
				NativeId:  "ETH",
				Threshold: 100,
			},
			{
				Name:      "SEI",
				Rpc:       "https://evm-rpc.sei-apis.com",
				NativeId:  "SEI",
				Threshold: 100,
			},
			{
				Name:      "Scroll",
				Rpc:       "https://scroll-mainnet.chainstacklabs.com",
				NativeId:  "ETH",
				Threshold: 100,
			},
			{
				Name:      "Kroma",
				Rpc:       "https://api.kroma.network",
				NativeId:  "ETH",
				Threshold: 100,
			},
			{
				Name:      "Taiko",
				Rpc:       "https://api.taikoscan.io/api",
				NativeId:  "ETH",
				Threshold: 100,
			},
			{
				Name:      "Optimism",
				Rpc:       "https://api-optimistic.etherscan.io/api",
				NativeId:  "ETH",
				Threshold: 100,
			},
			{
				Name:      "Linea",
				Rpc:       "https://rpc.linea.build",
				NativeId:  "ETH",
				Threshold: 100,
			},
			{
				Name:      "Rari",
				Rpc:       "https://rari.calderachain.xyz/infra-partner-http",
				NativeId:  "ETH",
				Threshold: 100,
			},
			{
				Name:      "Mint",
				Rpc:       "https://asia.rpc.mintchain.io",
				NativeId:  "ETH",
				Threshold: 100,
			},
			{
				Name:      "Polygon",
				Rpc:       "https://polygon.llamarpc.com",
				NativeId:  "POL",
				Threshold: 400,
			},
		}

		OPTIONS_INTERVALS = map[uint32]bool{
			1:  true,
			2:  true,
			5:  true,
			30: true,
		}

		SYMBOL_TO_OSTRICH_SYMBOL = map[string]string{
			"BTC":    "BTC_OSTRICH",
			"ETH":    "ETH_OSTRICH",
			"SOL":    "SOL_OSTRICH",
			"XAUT":   "XAUT_OSTRICH",
			"JPY":    "JPY_OSTRICH",
			"EUR":    "EUR_OSTRICH",
			"GBP":    "GBP_OSTRICH",
			"COIN":   "COIN_OSTRICH",
			"GOOGL":  "GOOGL_OSTRICH",
			"MSFT":   "MSFT_OSTRICH",
			"AMZN":   "AMZN_OSTRICH",
			"META":   "META_OSTRICH",
			"MSTR":   "MSTR_OSTRICH",
			"DEFT":   "DEFT_OSTRICH",
			"PLTR":   "PLTR_OSTRICH",
			"HOOD":   "HOOD_OSTRICH",
			"PYPL":   "PYPL_OSTRICH",
			"GME":    "GME_OSTRICH",
			"RIOT":   "RIOT_OSTRICH",
			"COKE":   "COKE_OSTRICH",
			"CRCL":   "CRCL_OSTRICH",
			"RDDT":   "RDDT_OSTRICH",
			"USOIL":  "USOIL_OSTRICH",
			"COPPER": "COPPER_OSTRICH",
			"NGAS":   "NGAS_OSTRICH",
			"Silver": "Silver_OSTRICH",
			"HIMS":   "HIMS_OSTRICH",
			"SPY":    "SPY_OSTRICH",
			"NA":     "NA_OSTRICH",
			"HUT":    "HUT_OSTRICH",
			"BRK.B":  "BRK_B_OSTRICH",
			"TCEHY":  "TCEHY_OSTRICH",
			"BIDU":   "BIDU_OSTRICH",
			"BABA":   "BABA_OSTRICH",
			"NKE":    "NKE_OSTRICH",
			"SOFI":   "SOFI_OSTRICH",
			"NIO":    "NIO_OSTRICH",
			"JD":     "JD_OSTRICH",
			"TRY":    "TRY_OSTRICH",
			"AUD":    "AUD_OSTRICH",
			"NZD":    "NZD_OSTRICH",
			"CHF":    "CHF_OSTRICH",
			"YHC":    "YHC_OSTRICH",
			"TRON":   "TRON_OSTRICH",
			"SONY":   "SONY_OSTRICH",
			"MTPLF":  "MTPLF_OSTRICH",
			"MUFG":   "MUFG_OSTRICH",
			"TM":     "TM_OSTRICH",
			"HMC":    "HMC_OSTRICH",
			"AZN":    "AZN_OSTRICH",
			"HSBC":   "HSBC_OSTRICH",
			"SHEL":   "SHEL_OSTRICH",
			"BCS":    "BCS_OSTRICH",
			"RYCEY":  "RYCEY_OSTRICH",
			"RICH":   "RICH_OSTRICH",
		}

		PRODUCT_ID_TO_MAILBOX_ADDRESS = make(map[uint32]string)
		for productId := range PRODUCT_ID_TO_SOURCE_CHAIN_INFO {
			PRODUCT_ID_TO_MAILBOX_ADDRESS[productId] = LOGX_MAILBOX_ADDRESS
		}
		PRODUCT_ID_TO_MAILBOX_ADDRESS[74] = OSTRICH_MAILBOX_ADDRESS

	default:
		log.Fatal("ENV not set")
	}
	// RPC URLs carry access tokens, so they are read from the environment rather than source.
	if rpcURL := os.Getenv("LOGX_RPC_URL"); rpcURL != "" {
		RPC_URL = rpcURL
	}
	if RPC_URL == "" {
		log.Println("WARNING: LOGX_RPC_URL not set; on-chain calls will fail")
	}
}
