package config

import (
	"log"
	"os"
)

const (
	TESTNET = "TESTNET"
	MAINNET = "MAINNET"
)

var MarketIdToSize map[uint]float64

var MarketIdToSlippage map[uint]float64

func Init() {
	switch os.Getenv("ENV") {
	case TESTNET:
		MarketIdToSize = map[uint]float64{
			1:   66,
			3:   4,
			5:   1400,
			7:   25000000,
			9:   30000,
			11:  40000,
			13:  40000,
			15:  80000,
			17:  10000,
			19:  80000,
			21:  10000,
			23:  90000, // as of 26 Oct 2024 price of EIGEN is ~2.7 USDC
			25:  20000,
			27:  2000000, // as of 21 Oct 2024 price of GOAT is ~0.35 USDC
			29:  800000,  // as of 21 Oct 2024 price of SPX is ~0.71 USDC
			31:  200000,
			33:  100000,
			35:  200000,
			37:  30000000,
			39:  5000000, //kPEPE
			41:  2500000, //kSHIB
			43:  4000000, // as of 26 OCT 2024 price of FARTCOIN is ~0.05 USDC
			45:  800000,  // as of 26 OCT 2024 price of MOODENG is ~0.25 USDC
			47:  1000000, // as of 28 OCT 2024 price of GIGA is ~0.05 USDC
			49:  2000000, // as of 28 OCT 2024 price of kBONK is ~0.02 USDC
			51:  400000,  // as of 28 OCT 2024 price of BRETT is ~0.1 USDC
			53:  500000,  // as of 28 OCT 2024 price of GRASS is ~0.8 USDC
			55:  100000,  // as of 30 OCT 2024 price of TIA is ~5.1 USDC
			57:  1000000, // as of 30 OCT 2024 price of SEI is ~0.41 USDC
			59:  200000,  // as of 30 OCT 2024 price of SUI is ~2.08 USDC
			61:  140000,  // as of 30 OCT 2024 price of ZRO is ~3.63 USDC
			63:  320000,  // as of 30 OCT 2024 price of TAIKO is ~1.43 USDC
			65:  100000,  // as of 30 OCT 2024 price of VIRTUAL is ~0.4247 USDC
			67:  5600000, // as of 30 OCT 2024 price of BOME is ~0.00776 USDC
			69:  8200,    // as of 30 OCT 2024 price of APE is ~0.93 USDC
			71:  400000,  // as of 30 OCT 2024 price of PONKE is ~0.4 USDC
			73:  440100,  // as of 30 OCT 2024 price of NPC is ~0.28 USDC
			75:  1200000, // as of 30 OCT 2024 price of SWELL is ~0.065 USDC
			77:  3000000, // as of 30 OCT 2024 price of JPY is ~0.0065 USDC
			79:  500000,  // as of 30 OCT 2024 price of ALGO is ~0.05 USDC
			81:  500000,  // as of 30 OCT 2024 price of XLM is ~0.05 USDC
			83:  500000,  // as of 30 OCT 2024 price of HBAR is ~0.05 USDC
			85:  9000,    // as of 30 OCT 2024 price of EUR is ~1.05 USDC
			87:  50,      // as of 30 OCT 2024 price of XAUT is ~2,643 USDC
			89:  80000,   // as of 30 OCT 2024 price of MOVE is ~0.68 USDC
			91:  800000,  // as of 30 OCT 2024 price of LOGX is ~0.052 USDC
			93:  5000,    // as of 30 OCT 2024 price of OFFICIAL TRUMP is ~30 USDC
			95:  30000,   // as of 27 MAR 2025 price of WALRUS is ~0.4 USDC = 30000 * 0.4 = 12000 USDC
			97:  3000,    // as of 12 MAY 2025 price of FLUID is ~4.9 USDC = 3000 * 5 = 15000 USDC
			99:  30000,   // as of 12 MAY 2025 price of PROMPT is ~0.35 USDC = 3000 * 0.35 = 10500 USDC
			101: 4000,    // as of 12 MAY 2025 price of NXPC is ~0.04 USDC = 300 * 3 = 12000 USDC
			103: 150,     // as of 12 MAY 2025 price of TSLA is ~340 USDC = 150 * 340 = 51000 USDC
			105: 500,     // as of 12 MAY 2025 price of NVDA is ~130 USDC = 500 * 130 = 65000 USDC
			107: 200,     // as of 12 MAY 2025 price of AAPL is ~200 USDC = 200 * 200 = 40000 USDC
			109: 100,     // as of 26 MAY 2025 price of QQQ is ~200 USDC = 100 * 500 = 40000 USDC
			111: 66,
			113: 4,
			115: 1400,
			117: 50,
			119: 3000000,
			121: 9000,
			123: 7000,
			125: 200,     // as of 2 JUNE 2025 price of COIN is ~246 USDC = 200 * 246 = 49200 USDC
			127: 300,     // as of 2 JUNE 2025 price of GOOGL is ~168 USDC = 300 * 168 = 50400 USDC
			129: 110,     // as of 2 JUNE 2025 price of MSFT is ~461 USDC = 110 * 461 = 50710 USDC
			131: 240,     // as of 2 JUNE 2025 price of AMZN is ~205 USDC = 240 * 205 = 49200 USDC
			133: 80,      // as of 2 JUNE 2025 price of META is ~662 USDC = 80 * 662 = 52960 USDC
			135: 150,     // as of 2 JUNE 2025 price of MSTR is ~376 USDC = 150 * 376 = 56400 USDC
			137: 14000,   // as of 2 JUNE 2025 price of DEFT is ~3.55 USDC = 14000 * 3.55 = 49700 USDC
			139: 400,     // as of 2 JUNE 2025 price of PLTR is ~131 USDC = 400 * 131 = 52400 USDC
			141: 700,     // as of 2 JUNE 2025 price of HOOD is ~70 USDC = 700 * 70 = 49000 USDC
			143: 700,     // as of 2 JUNE 2025 price of PYPL is ~70 USDC = 700 * 70 = 49000 USDC
			145: 1650,    // as of 2 JUNE 2025 price of GME is ~30 USDC = 1650 * 30 = 49500 USDC
			147: 6250,    // as of 2 JUNE 2025 price of RIOT is ~8 USDC = 6250 * 8 = 50000 USDC
			149: 700,     // as of 2 JUNE 2025 price of COKE is ~71 USDC = 700 * 71 = 49700 USDC
			151: 450,     // as of 2 JUNE 2025 price of CRCL_OSTRICH is ~107 USDC = 450 * 107 = 48150 USDC
			153: 350,     // as of 2 JUNE 2025 price of RDDT_OSTRICH is ~138 USDC = 350 * 138 = 48300 USDC
			155: 700,     // as of 2 JUNE 2025 price of USOIL_OSTRICH is ~73 USDC = 700 * 73 = 50000 USDC
			157: 9000,    // as of 2 JUNE 2025 price of COPPER_OSTRICH is ~5 USDC = 9000 * 5 = 45000 USDC
			159: 13000,   // as of 2 JUNE 2025 price of NGAS_OSTRICH is ~3.7 USDC = 13000 * 3.7 = 48100 USDC
			161: 1400,    // as of 2 JUNE 2025 price of Silver_OSTRICH is ~36 USDC = 1400 * 36 = 50400 USDC
			163: 1400,    // as of 2 JUNE 2025 price of HIMS_OSTRICH is ~35 USDC = 1400 * 35 = 49000 USDC
			165: 80,      // as of 2 JUNE 2025 price of SPY_OSTRICH is ~600 USDC = 80 * 600 = 48000 USDC
			167: 5000,    // as of 2 JUNE 2025 price of NA_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			169: 4000,    // as of 2 JUNE 2025 price of HUT_OSTRICH is ~15 USDC = 4000 * 15 = 60000 USDC
			171: 100,     // as of 2 JUNE 2025 price of BRK_B_OSTRICH is ~485 USDC = 100 * 485 = 48500 USDC
			173: 700,     // as of 2 JUNE 2025 price of TCEHY_OSTRICH is ~67 USDC = 700 * 67 = 46900 USDC
			175: 500,     // as of 2 JUNE 2025 price of BIDU_OSTRICH is ~86 USDC = 500 * 86 = 43000 USDC
			177: 400,     // as of 2 JUNE 2025 price of BABA_OSTRICH is ~110 USDC = 400 * 110 = 44000 USDC
			179: 400,     // as of 2 JUNE 2025 price of NKE_OSTRICH is ~80 USDC = 400 * 80 = 32000 USDC
			181: 1800,    // as of 2 JUNE 2025 price of SOFI_OSTRICH is ~20 USDC = 1800 * 20 = 36000 USDC
			183: 4000000, // as of 2 JUNE 2025 price of PUMP is ~0.005 USDC = 4000000 * 0.005 = 10000 USDC
			185: 12000,   //as of 15 JULY 2025 price of NIO_OSTRICH is ~4.17 USDC = 12000 * 4.17 = 50040 USDC
			187: 1500,    // as of 15 JULY 2025 price of JD_OSTRICH is ~30.8 USDC = 1500 * 30.8 = 46200 USDC
			189: 1000000, // as of 15 JULY 2025 price of TRY_OSTRICH is ~0.02 USDC = 1000000 * 0.02 = 20000 USDC
			191: 80000,   // as of 15 JULY 2025 price of AUD_OSTRICH is ~0.65 USDC = 80000 * 0.65 = 52000 USDC
			193: 80000,   // as of 15 JULY 2025 price of NZD_OSTRICH is ~0.59 USDC = 80000 * 0.59 = 47200 USDC
			195: 40000,   // as of 15 JULY 2025 price of CHF_OSTRICH is ~1.2 USDC = 40000 * 1.2 = 48000 USDC
			197: 6250,    // as of 15 JULY 2025 price of YHC_OSTRICH is ~ 8 USDC = 6250 * 8 = 50000 USDC
			199: 5200,    // as of 15 JULY 2025 price of TRON_OSTRICH is ~9.5 USDC = 5200 * 9.5 = 49400 USDC
			201: 5000,    // as of 15 JULY 2025 price of SONY_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			203: 5000,    // as of 15 JULY 2025 price of MTPLF_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			205: 5000,    // as of 15 JULY 2025 price of MUFG_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			207: 5000,    // as of 15 JULY 2025 price of TM_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			209: 5000,    // as of 15 JULY 2025 price of HMC_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			211: 5000,    // as of 15 JULY 2025 price of AZN_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			213: 5000,    // as of 15 JULY 2025 price of HSBC_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			215: 5000,    // as of 15 JULY 2025 price of SHEL_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			217: 5000,    // as of 15 JULY 2025 price of BCS_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			219: 5000,    // as of 15 JULY 2025 price of RYCEY_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			221: 10000,   // as of 15 JULY 2025 price of RICH_OSTRICH is ~3 USDC = 10000 * 3 = 30000 USDC
		}
		MarketIdToSlippage = map[uint]float64{
			1:   0.0005,
			3:   0.0005,
			5:   0.0005,
			7:   0.001,
			9:   0.005,
			11:  0.005,
			13:  0.005,
			15:  0.005,
			17:  0.005,
			19:  0.005,
			21:  0.005,
			23:  0.003,
			25:  0.003,
			27:  0.003,
			29:  0.003,
			31:  0.003,
			33:  0.003,
			35:  0.003,
			37:  0.003,
			39:  0.003,
			41:  0.005,
			43:  0.005,
			45:  0.005,
			47:  0.005,
			49:  0.008,
			51:  0.008,
			53:  0.008,
			55:  0.008,
			57:  0.008,
			59:  0.008,
			61:  0.008,
			63:  0.008,
			65:  0.008,
			67:  0.008,
			69:  0.008,
			71:  0.008,
			73:  0.008,
			75:  0.008,
			77:  0.008,
			79:  0.008,
			81:  0.008,
			83:  0.008,
			85:  0.008,
			87:  0.008,
			89:  0.01,
			91:  0.01,
			93:  0.01,
			95:  0.005,
			97:  0.005,
			99:  0.005,
			101: 0.005,
			103: 0.0003,
			105: 0.0003,
			107: 0.0003,
			109: 0.0003,
			111: 0.0003,
			113: 0.0003,
			115: 0.0003,
			117: 0.0003,
			119: 0.0003,
			121: 0.0003,
			123: 0.0003,
			125: 0.0003,
			127: 0.0003,
			129: 0.0003,
			131: 0.0003,
			133: 0.0003,
			135: 0.0003,
			137: 0.0003,
			139: 0.0003,
			141: 0.0003,
			143: 0.0003,
			145: 0.0003,
			147: 0.0003,
			149: 0.0003,
			151: 0.0035,
			153: 0.0003,
			155: 0.0005,
			157: 0.005,
			159: 0.0005,
			161: 0.0005,
			163: 0.0008,
			165: 0.0008,
			167: 0.001,
			169: 0.001,
			171: 0.001,
			173: 0.001,
			175: 0.001,
			177: 0.001,
			179: 0.001,
			181: 0.001,
			183: 0.003,
			185: 0.001,
			187: 0.001,
			189: 0.001,
			191: 0.001,
			193: 0.001,
			195: 0.001,
			197: 0.001,
			199: 0.001,
			201: 0.001,
			203: 0.001,
			205: 0.001,
			207: 0.001,
			209: 0.001,
			211: 0.001,
			213: 0.001,
			215: 0.001,
			217: 0.001,
			219: 0.001,
			221: 0.01,
		}

	case MAINNET:
		MarketIdToSize = map[uint]float64{
			1:   66,
			3:   10,
			5:   1000,
			7:   25000000,
			9:   30000,
			11:  40000,
			13:  40000,
			15:  80000,
			17:  10000,
			19:  80000,
			21:  10000,
			23:  90000, // as of 26 Oct 2024 price of EIGEN is ~2.7 USDC
			25:  20000,
			27:  2000000, // as of 21 Oct 2024 price of GOAT is ~0.35 USDC
			29:  800000,  // as of 21 Oct 2024 price of SPX is ~0.71 USDC
			31:  200000,
			33:  100000,
			35:  200000,
			37:  150000000,
			39:  5000000, //kPEPE
			41:  2500000, //kSHIB
			43:  4000000, // as of 26 OCT 2024 price of FARTCOIN is ~0.05 USDC
			45:  800000,  // as of 26 OCT 2024 price of MOODENG is ~0.25 USDC
			47:  1000000, // as of 28 OCT 2024 price of GIGA is ~0.05 USDC
			49:  2000000, // as of 28 OCT 2024 price of kBONK is ~0.02 USDC
			51:  400000,  // as of 28 OCT 2024 price of BRETT is ~0.1 USDC
			53:  100000,  // as of 28 OCT 2024 price of GRASS is ~0.8 USDC
			55:  50000,   // as of 30 OCT 2024 price of TIA is ~5.1 USDC
			57:  100000,  // as of 30 OCT 2024 price of SEI is ~0.41 USDC
			59:  100000,  // as of 30 OCT 2024 price of SUI is ~2.08 USDC
			61:  140000,  // as of 30 OCT 2024 price of ZRO is ~3.63 USDC
			63:  160000,  // as of 30 OCT 2024 price of TAIKO is ~1.43 USDC
			65:  100000,  // as of 30 OCT 2024 price of VIRTUAL is ~0.4247 USDC
			67:  5600000, // as of 30 OCT 2024 price of BOME is ~0.00776 USDC
			69:  50000,   // as of 30 OCT 2024 price of APE is ~0.93 USDC
			71:  400000,  // as of 30 OCT 2024 price of PONKE is ~0.4 USDC
			73:  440100,  // as of 30 OCT 2024 price of NPC is ~0.28 USDC
			75:  4000000, // as of 30 OCT 2024 price of SWELL is ~0.065 USDC
			77:  3000000, // as of 30 OCT 2024 price of JPY is ~0.0065 USDC
			79:  250000,  // as of 30 OCT 2024 price of ALGO is ~0.05 USDC
			81:  500000,  // as of 30 OCT 2024 price of XLM is ~0.05 USDC
			83:  500000,  // as of 30 OCT 2024 price of HBAR is ~0.05 USDC
			85:  9000,    // as of 30 OCT 2024 price of EUR is ~1.05 USDC
			87:  50,      // as of 30 OCT 2024 price of XAUT is ~2,643 USDC
			89:  80000,   // as of 30 OCT 2024 price of MOVE is ~0.68 USDC
			91:  800000,  // as of 30 OCT 2024 price of LOGX is ~0.052 USDC
			93:  5000,    // as of 30 OCT 2024 price of OFFICIAL TRUMP is ~30 USDC
			95:  30000,   // as of 27 MAR 2025 price of WALRUS is ~0.4 USDC = 30000 * 0.4 = 12000 USDC
			97:  3000,    // as of 12 MAY 2025 price of FLUID is ~4.97 USDC = 3000 * 5 = 15000 USDC
			99:  30000,   // as of 12 MAY 2025 price of PROMPT is ~0.35 USDC = 30000 * 0.35 = 10500 USDC
			101: 4000,    // as of 12 MAY 2025 price of NXPC is ~0.04 USDC = 4000 * 2.5 = 10000 USDC
			103: 150,     // as of 12 MAY 2025 price of TSLA is ~340 USDC = 150 * 340 = 51000 USDC
			105: 500,     // as of 12 MAY 2025 price of NVDA is ~130 USDC = 500 * 130 = 65000 USDC
			107: 200,     // as of 12 MAY 2025 price of AAPL is ~200 USDC = 200 * 200 = 40000 USDC
			109: 100,     // as of 12 MAY 2025 price of QQQ is ~200 USDC = 100 * 500 = 40000 USDC
			111: 6,
			113: 100,
			115: 2000,
			117: 50,
			119: 3000000,
			121: 9000,
			123: 7000,    // as of 27 MAY 2025 price of GBP is ~1.3 USDC = 7000 * 1.3 = 9100 USDC
			125: 200,     // as of 2 JUNE 2025 price of COIN is ~246 USDC = 200 * 246 = 49200 USDC
			127: 300,     // as of 2 JUNE 2025 price of GOOGL is ~168 USDC = 300 * 168 = 50400 USDC
			129: 110,     // as of 2 JUNE 2025 price of MSFT is ~461 USDC = 110 * 461 = 50710 USDC
			131: 240,     // as of 2 JUNE 2025 price of AMZN is ~205 USDC = 240 * 205 = 49200 USDC
			133: 80,      // as of 2 JUNE 2025 price of META is ~662 USDC = 80 * 662 = 52960 USDC
			135: 150,     // as of 2 JUNE 2025 price of MSTR is ~376 USDC = 150 * 376 = 56400 USDC
			137: 14000,   // as of 2 JUNE 2025 price of DEFT is ~3.55 USDC = 14000 * 3.55 = 49700 USDC
			139: 400,     // as of 2 JUNE 2025 price of PLTR is ~131 USDC = 400 * 131 = 52400 USDC
			141: 700,     // as of 2 JUNE 2025 price of HOOD is ~70 USDC = 700 * 70 = 49000 USDC
			143: 700,     // as of 2 JUNE 2025 price of PYPL is ~70 USDC = 700 * 70 = 49000 USDC
			145: 1650,    // as of 2 JUNE 2025 price of GME is ~30 USDC = 1650 * 30 = 49500 USDC
			147: 625,     // as of 2 JUNE 2025 price of RIOT is ~8 USDC = 6250 * 8 = 50000 USDC
			149: 700,     // as of 2 JUNE 2025 price of COKE is ~71 USDC = 700 * 71 = 49700 USDC
			151: 50,      // as of 2 JUNE 2025 price of CRCL_OSTRICH is ~107 USDC = 450 * 107 = 48150 USDC
			153: 300,     // as of 2 JUNE 2025 price of RDDT_OSTRICH is ~138 USDC = 350 * 138 = 48300 USDC
			155: 350,     // as of 2 JUNE 2025 price of USOIL_OSTRICH is ~73 USDC = 700 * 73 = 50000 USDC
			157: 9000,    // as of 2 JUNE 2025 price of COPPER_OSTRICH is ~5 USDC = 9000 * 5 = 45000 USDC
			159: 8000,    // as of 2 JUNE 2025 price of NGAS_OSTRICH is ~3.7 USDC = 13000 * 3.7 = 48100 USDC
			161: 1400,    // as of 2 JUNE 2025 price of Silver_OSTRICH is ~36 USDC = 1400 * 36 = 50400 USDC
			163: 1400,    // as of 2 JUNE 2025 price of HIMS_OSTRICH is ~35 USDC = 1400 * 35 = 49000 USDC
			165: 80,      // as of 2 JUNE 2025 price of SPY_OSTRICH is ~600 USDC = 80 * 600 = 48000 USDC
			167: 1000,    // as of 2 JUNE 2025 price of NA_OSTRICH is ~10 USDC = 5000 * 10 = 50000 USDC
			169: 2000,    // as of 2 JUNE 2025 price of HUT_OSTRICH is ~15 USDC = 4000 * 15 = 60000 USDC
			171: 130,     // as of 30 JUNE 2025 price of BRK_B_OSTRICH is ~485 USDC = 100 * 485 = 48500 USDC
			173: 1000,    // as of 2 JUNE 2025 price of TCEHY_OSTRICH is ~67 USDC = 700 * 67 = 46900 USDC
			175: 1000,    // as of 2 JUNE 2025 price of BIDU_OSTRICH is ~86 USDC = 500 * 86 = 43000 USDC
			177: 1000,    // as of 2 JUNE 2025 price of BABA_OSTRICH is ~110 USDC = 400 * 110 = 44000 USDC
			179: 400,     // as of 2 JUNE 2025 price of NKE_OSTRICH is ~80 USDC = 400 * 80 = 32000 USDC
			181: 1800,    // as of 2 JUNE 2025 price of SOFI_OSTRICH is ~20 USDC = 1800 * 20 = 36000 USDC
			183: 4000000, // as of 2 JUNE 2025 price of PUMP is ~0.005 USDC = 4000000 * 0.005 = 20000 USDC
			185: 12000,   //as of 15 JULY 2025 price of NIO_OSTRICH  is ~4.17 USDC = 12000 * 4.17 = 50040 USDC
			187: 1500,    // as of 15 JULY 2025 price of JD_OSTRICH is ~30.8 USDC = 1500 * 30.8 = 46200 USDC
			189: 1000000, // as of 15 JULY 2025 price of TRY_OSTRICH is ~0.02 USDC = 1000000 * 0.02 = 20000 USDC
			191: 80000,   // as of 15 JULY 2025 price of AUD_OSTRICH is ~0.65 USDC = 80000 * 0.65 = 52000 USDC
			193: 80000,   // as of 15 JULY 2025 price of NZD_OSTRICH is ~0.59 USDC = 80000 * 0.59 = 47200 USDC
			195: 40000,   // as of 15 JULY 2025 price of CHF_OSTRICH is ~1.2 USDC = 40000 * 1.2 = 48000 USDC
			197: 6250,    // as of 15 JULY 2025 price of YHC_OSTRICH is ~ 8 USDC = 6250 * 8 = 50000 USDC
			199: 5200,    // as of 15 JULY 2025 price of TRON_OSTRICH is ~9.5 USDC = 5200 * 9.5 = 49400 USDC
			201: 2000,    // as of 15 JULY 2025 price of SONY_OSTRICH is ~24 USDC = 2000 * 24 = 48000 USDC
			203: 6000,    // as of 15 JULY 2025 price of MTPLF_OSTRICH is ~7 USDC = 6000 * 7 = 42000 USDC
			205: 3000,    // as of 15 JULY 2025 price of MUFG_OSTRICH is ~14 USDC = 3000 * 14 = 42000 USDC
			207: 250,     // as of 15 JULY 2025 price of TM_OSTRICH is ~184 USDC = 250 * 184 = 46000 USDC
			209: 1400,    // as of 15 JULY 2025 price of HMC_OSTRICH is ~32 USDC = 1400 * 32 = 44800 USDC
			211: 100,     // as of 15 JULY 2025 price of AZN_OSTRICH is ~625 USDC = 100 * 625 = 62500 USDC
			213: 1000,    // as of 15 JULY 2025 price of HSBC_OSTRICH is ~65 USDC = 1000 * 65 = 65000 USDC
			215: 1000,    // as of 15 JULY 2025 price of SHEL_OSTRICH is ~73 USDC = 1000 * 73 = 73000 USDC
			217: 2000,    // as of 15 JULY 2025 price of BCS_OSTRICH is ~20 USDC = 2000 * 20 = 40000 USDC
			219: 3000,    // as of 15 JULY 2025 price of RYCEY_OSTRICH is ~14 USDC = 3000 * 14 = 42000 USDC
			221: 10000,   // as of 15 JULY 2025 price of RICH_OSTRICH is ~3 USDC = 10000 * 3 = 30000 USDC
		}
		MarketIdToSlippage = map[uint]float64{
			1:   0.0008,
			3:   0.0008,
			5:   0.0008,
			7:   0.001,
			9:   0.0025,
			11:  0.0025,
			13:  0.0025,
			15:  0.0025,
			17:  0.0025,
			19:  0.0025,
			21:  0.0025,
			23:  0.0025,
			25:  0.0025,
			27:  0.001,
			29:  0.001,
			31:  0.001,
			33:  0.001,
			35:  0.001,
			37:  0.001,
			39:  0.001,
			41:  0.002,
			43:  0.0025,
			45:  0.0025,
			47:  0.0025,
			49:  0.0025,
			51:  0.0025,
			53:  0.0025,
			55:  0.0025,
			57:  0.0025,
			59:  0.0025,
			61:  0.0025,
			63:  0.0025,
			65:  0.0025,
			67:  0.0025,
			69:  0.0025,
			71:  0.0025,
			73:  0.0025,
			75:  0.0025,
			77:  0.0025,
			79:  0.0025,
			81:  0.0025,
			83:  0.0025,
			85:  0.0025,
			87:  0.0025,
			89:  0.008,
			91:  0.03,
			93:  0.0025,
			95:  0.005,
			97:  0.005,
			99:  0.005,
			101: 0.01,
			103: 0.01,
			105: 0.01,
			107: 0.01,
			109: 0.01,
			111: 0.0005,
			113: 0.0005,
			115: 0.0005,
			117: 0.01,
			119: 0.01,
			121: 0.01,
			123: 0.01,
			125: 0.01,
			127: 0.01,
			129: 0.01,
			131: 0.01,
			133: 0.01,
			135: 0.01,
			137: 0.01,
			139: 0.03,
			141: 0.01,
			143: 0.01,
			145: 0.01,
			147: 0.20,
			149: 0.01,
			151: 0.10,
			153: 0.01,
			155: 0.01,
			157: 0.01,
			159: 0.01,
			161: 0.01,
			163: 0.04,
			165: 0.01,
			167: 0.05,
			169: 0.01,
			171: 0.01,
			173: 0.01,
			175: 0.01,
			177: 0.01,
			179: 0.01,
			181: 0.002,
			183: 0.003,
			185: 0.01,
			187: 0.01,
			189: 0.01,
			191: 0.01,
			193: 0.01,
			195: 0.01,
			197: 0.01,
			199: 0.01,
			201: 0.01,
			203: 0.01,
			205: 0.01,
			207: 0.01,
			209: 0.01,
			211: 0.01,
			213: 0.01,
			215: 0.01,
			217: 0.01,
			219: 0.01,
			221: 0.01,
		}

	default:
		log.Fatal("ENV not set")
	}
}
