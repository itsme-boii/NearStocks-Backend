import { Injectable, OnModuleDestroy } from '@nestjs/common';
import axios from 'axios';
import { RedisService } from './redis/redis.service';
import { WinstonLogger } from './utils/winston.service';
import { BigNumber } from 'ethers';
import { NUMERICAL_STOCK_TICKER_MAP, HYPERLIQUID_TOKEN_MAP } from './config/numericalStockTicker.config';
import { normalizeForexSymbol, parseAndNormalizeForexPairs } from './utils/forex.utils';
import { CantonPricePillService } from './canton/canton-pricepill.service';

type PriceData = {
  USDC?: number;
  ETH?: number;
  BTC?: number;
};

@Injectable()
export class PriceService implements OnModuleDestroy {
  private logger: WinstonLogger;
  private supportedTokens: any;
  private allSupportedTokens: any;
  private exchangeWeights: any;
  private debug;
  private PRICE_PRECISION: BigNumber = BigNumber.from('10000000000000000000000000000'); //10^28 as this provides 2 digit precision
  private finnhubApiKeys: string[] = [];
  private currentKeyIndex = 0;
  private cachedStockPrices: Record<string, { price: number; timestamp: number }> = {};
  private finnhubFetchInterval: NodeJS.Timeout | null = null;

  private allticksApiKeys: string[] = [];
  private currentAllticksKeyIndex = 0;
  private cachedAllticksStockPrices: Record<string, { price: number; timestamp: number }> = {};
  private cachedAllticksCommodityPrices: Record<string, { price: number; timestamp: number }> = {};
  private allticksFetchInterval: NodeJS.Timeout | null = null;

  private tiingoApiToken: string = '';
  private cachedTiingoStockPrices: Record<string, { price: number; timestamp: number }> = {};
  private tiingoFetchInterval: NodeJS.Timeout | null = null;

  private cachedTiingoForexPrices: Record<string, { price: number; timestamp: number }> = {};
  private tiingoForexFetchInterval: NodeJS.Timeout | null = null;

  // Hyperliquid API properties
  private hyperliquidTokens: string[] = [];

  // Canton PricePill properties
  private cachedCantonPrices: Record<string, number> = {};
  private cantonSyncInterval: NodeJS.Timeout | null = null;
  private cantonRetryTimeout: NodeJS.Timeout | null = null;
  private cantonPrimaryTokens: Set<string> = new Set();

  constructor(
    private readonly redisService: RedisService,
    private readonly cantonPricePillService: CantonPricePillService,
  ) {
    this.logger = new WinstonLogger('PriceService');
    this.logger.log('PriceService started');
    const tokens: string = process.env.SUPPORTED_TOKENS;
    this.supportedTokens = tokens.split(',');
    this.allSupportedTokens = process.env.ALL_SUPPORTED_TOKENS.split(',');

    const finnhubStocks = this.parseEnvList(process.env.FINNHUB_STOCKS);
    this.allSupportedTokens.push(...finnhubStocks);

    const tiingoStocks = this.parseEnvList(process.env.TIINGO_STOCKS);
    this.allSupportedTokens.push(...tiingoStocks);

    // Parse and normalize forex pairs from environment variable
    const normalizedForexPairs = parseAndNormalizeForexPairs(process.env.TIINGO_FOREX_PAIRS);
    this.allSupportedTokens.push(...normalizedForexPairs);

    // TODO: Move this to env in ALL_SUPPORTED_TOKENS and SUPPORTED_TOKENS
    const allticksStocks = this.parseEnvList(process.env.ALLTICKS_STOCKS);
    const transformedAllticksStocks = allticksStocks.map((stock) => {
      return NUMERICAL_STOCK_TICKER_MAP[stock] || stock;
    });

    this.allSupportedTokens.push(...transformedAllticksStocks);

    // TODO: Move this to env in ALL_SUPPORTED_TOKENS and SUPPORTED_TOKENS
    const allticksCommodities = this.parseEnvList(process.env.ALLTICKS_COMMODITIES);
    this.allSupportedTokens.push(...allticksCommodities);

    // Add Hyperliquid tokens to supported tokens
    this.hyperliquidTokens = this.parseEnvList(process.env.HYPERLIQUID_TOKENS);
    const mappedHyperliquidTokens = this.hyperliquidTokens.map((token) => HYPERLIQUID_TOKEN_MAP[token] || token);
    this.allSupportedTokens.push(...mappedHyperliquidTokens);

    // Add Canton PricePill feed IDs to supported tokens
    const cantonFeedIds = this.parseEnvList(process.env.CANTON_FEED_IDS);
    this.allSupportedTokens.push(...cantonFeedIds);

    this.allSupportedTokens = [...new Set(this.allSupportedTokens)];

    this.finnhubApiKeys = this.parseEnvList(process.env.FINNHUB_API_KEYS);

    if (this.finnhubApiKeys.length === 0 && process.env.FINNHUB_API_KEY) {
      this.finnhubApiKeys.push(process.env.FINNHUB_API_KEY);
    }

    this.logger.log(`Loaded ${this.finnhubApiKeys.length} Finnhub API keys`);

    this.allticksApiKeys = this.parseEnvList(process.env.ALLTICKS_API_KEYS);

    if (this.allticksApiKeys.length === 0 && process.env.ALLTICKS_API_KEY) {
      this.allticksApiKeys.push(process.env.ALLTICKS_API_KEY);
    }

    this.tiingoApiToken = process.env.TIINGO_API_TOKEN || '';

    this.debug = Boolean(process.env.DEBUG_CONSOLE_LOGS);
    this.exchangeWeights = {
      RedStonePricePill: 7,
      Binance: 3,
      Coinbase: 2,
      OKX: 2,
      Bybit: 2,
      Kraken: 2,
      Htx: 1,
      RedStone: 1,
      Mexc: 1,
      Gate: 1,
      Finnhub: 1,
      Allticks: 1,
      Tiingo: 5,
      TiingoForex: 3,
      Hyperliquid: 1,
    };
    this.logger.log(`Supported Tokens: ${this.supportedTokens}`);
    this.logger.log(`All Supported Tokens: ${this.allSupportedTokens}`);
    this.logger.log(`Hyperliquid Tokens: ${this.hyperliquidTokens}`);
    this.startAfterCertainTime(this.getPriceForSupportedTokens.bind(this), 5000, Number(process.env.TIMEOUT));

    const finnhubStocksCount = finnhubStocks.length;
    const keysCount = this.finnhubApiKeys.length;

    if (finnhubStocksCount > 0 && keysCount > 0) {
      const intervalMs = Math.max(Number(process.env.FINNHUB_FETCH_INTERVAL || 30000), Math.ceil((finnhubStocksCount / keysCount) * 1000));

      this.logger.log(`Starting Finnhub fetcher with interval: ${intervalMs}ms for ${finnhubStocksCount} stocks using ${keysCount} keys`);

      setTimeout(() => {
        this.fetchFinnhubStocks();

        this.finnhubFetchInterval = setInterval(this.fetchFinnhubStocks.bind(this), intervalMs);
      }, 8000);
    }

    const allticksStocksCount = allticksStocks.length;
    const allticksKeysCount = this.allticksApiKeys.length;

    if (allticksStocksCount > 0 && allticksKeysCount > 0) {
      const allticksIntervalMs = Number(process.env.ALLTICKS_FETCH_INTERVAL || 10000);

      this.logger.log(`Starting Allticks fetcher with interval: ${allticksIntervalMs}ms for ${allticksStocksCount} stocks using ${allticksKeysCount} keys`);

      setTimeout(() => {
        this.fetchAllticksStocks();

        this.allticksFetchInterval = setInterval(this.fetchAllticksStocks.bind(this), allticksIntervalMs);
      }, 12000); // Start after Finnhub to avoid startup conflicts like increased cpu and memory usage
    }

    const allticksCommoditiesCount = allticksCommodities.length;

    if (allticksCommoditiesCount > 0 && allticksKeysCount > 0) {
      const allticksCommoditiesIntervalMs = Number(process.env.ALLTICKS_COMMODITIES_FETCH_INTERVAL || 10000);

      this.logger.log(`Starting Allticks commodities fetcher with interval: ${allticksCommoditiesIntervalMs}ms for ${allticksCommoditiesCount} commodities using ${allticksKeysCount} keys`);

      setTimeout(() => {
        this.fetchAllticksCommodities();

        this.allticksFetchInterval = setInterval(this.fetchAllticksCommodities.bind(this), allticksCommoditiesIntervalMs);
      }, 10000); // Start after stocks to avoid conflicts
    }

    const tiingoStocksCount = tiingoStocks.length;

    if (tiingoStocksCount > 0 && this.tiingoApiToken) {
      const tiingoIntervalMs = Number(process.env.TIINGO_FETCH_INTERVAL || 30000);

      this.logger.log(`Starting Tiingo fetcher with interval: ${tiingoIntervalMs}ms for ${tiingoStocksCount} stocks`);

      setTimeout(() => {
        this.fetchTiingoStocks();

        this.tiingoFetchInterval = setInterval(this.fetchTiingoStocks.bind(this), tiingoIntervalMs);
      }, 12000); // Start after Allticks to avoid conflicts
    }

    const tiingoForexPairsCount = this.parseEnvList(process.env.TIINGO_FOREX_PAIRS).length;

    if (tiingoForexPairsCount > 0 && this.tiingoApiToken) {
      const tiingoForexIntervalMs = Number(process.env.TIINGO_FOREX_FETCH_INTERVAL || 30000);

      this.logger.log(`Starting Tiingo Forex fetcher with interval: ${tiingoForexIntervalMs}ms for ${tiingoForexPairsCount} forex pairs`);

      setTimeout(() => {
        this.fetchTiingoForex();

        this.tiingoForexFetchInterval = setInterval(this.fetchTiingoForex.bind(this), tiingoForexIntervalMs);
      }, 12000); // Start after Tiingo stocks to avoid conflicts
    }

    if (cantonFeedIds.length > 0) {
      this.cantonPrimaryTokens = new Set(cantonFeedIds);
      this.logger.log(`Starting Canton PricePill streaming for ${cantonFeedIds.length} tokens (primary: ${cantonFeedIds.join(', ')})`);

      // Wait for a valid auth token before starting streaming and sync
      this.initCantonStreaming();
    }
  }

  onModuleDestroy() {
    if (this.finnhubFetchInterval) {
      clearInterval(this.finnhubFetchInterval);
      this.finnhubFetchInterval = null;
    }

    if (this.allticksFetchInterval) {
      clearInterval(this.allticksFetchInterval);
      this.allticksFetchInterval = null;
    }

    if (this.tiingoFetchInterval) {
      clearInterval(this.tiingoFetchInterval);
      this.tiingoFetchInterval = null;
    }

    if (this.tiingoForexFetchInterval) {
      clearInterval(this.tiingoForexFetchInterval);
      this.tiingoForexFetchInterval = null;
    }

    if (this.cantonSyncInterval) {
      clearInterval(this.cantonSyncInterval);
      this.cantonSyncInterval = null;
    }
    if (this.cantonRetryTimeout) {
      clearTimeout(this.cantonRetryTimeout);
      this.cantonRetryTimeout = null;
    }
    this.cantonPricePillService.stopStreaming();
  }

  private initCantonStreaming(): void {
    const MIN_SYNC_INTERVAL_MS = 1000;
    const parsed = Number(process.env.CANTON_PRICEPILL_FETCH_INTERVAL || 5000);
    const syncIntervalMs = Number.isFinite(parsed) && parsed >= MIN_SYNC_INTERVAL_MS ? parsed : 5000;

    this.cantonPricePillService
      .waitForAuth()
      .then(() => {
        this.logger.log('Canton auth token acquired, starting streaming and sync');
        this.cantonPricePillService.startStreaming().catch((err) => {
          this.logger.error('Canton streaming failed to start', err?.message || err);
        });
        this.cantonSyncInterval = setInterval(this.syncCantonPrices.bind(this), syncIntervalMs);
      })
      .catch((err) => {
        this.logger.error('Canton auth failed, retrying in 10s', err?.message || err);
        this.cantonRetryTimeout = setTimeout(() => this.initCantonStreaming(), 10_000);
      });
  }

  syncCantonPrices = () => {
    const prices = this.cantonPricePillService.getCachedPrices();
    const priceCount = Object.keys(prices).length;
    if (priceCount > 0) {
      this.cachedCantonPrices = prices;
      this.logger.log(`Canton PricePill prices synced: ${priceCount} entries`);
      if (this.debug) {
        this.logger.log(`Canton PricePill prices synced payload: ${JSON.stringify(prices)}`);
      }
    }
  };

  fetchFromCanton = async (): Promise<PriceData> => {
    // Always pull latest from the streaming service (covers first cycle before interval fires)
    this.syncCantonPrices();
    this.debug ? this.logger.log(`Canton Cached Prices: ${JSON.stringify(this.cachedCantonPrices)}`) : '';
    return { ...this.cachedCantonPrices };
  };

  sendToRedis = async (precisionPrice, token) => {
    try {
      await this.redisService.set(token, JSON.stringify(precisionPrice));
      this.logger.log(`Price of ${token} written to redis: ${precisionPrice.price.div(this.PRICE_PRECISION).toString()}`);
    } catch (error) {
      this.logger.log(`Error writting price of ${token} to redis`);
    }
  };

  findMedian = (arr: number[]): number => {
    try {
      const sortedArr = arr.slice().sort((a, b) => a - b);

      const isEvenLength = sortedArr.length % 2 === 0;

      if (isEvenLength) {
        const middleIndex = sortedArr.length / 2;
        const median = (Number(sortedArr[middleIndex - 1]) + Number(sortedArr[middleIndex])) / 2;
        return median;
      } else {
        const middleIndex = Math.floor(sortedArr.length / 2);
        return sortedArr[middleIndex];
      }
    } catch (error) {
      this.logger.error('Error while finding mediam price', error);
    }
  };

  convertToPrecisionFormat = (
    medianPrice: number,
  ): {
    price: BigNumber;
    expo: number;
    publishTime: number;
  } => {
    try {
      const priceString = medianPrice.toString();
      const parts = priceString.split('.');

      const price = parseInt(parts.join(''));
      const expo = parts.length > 1 ? parts[1].length : 0; // Count the number of decimal places
      const pendingExpo = BigNumber.from(10).pow(BigNumber.from(30 - expo));
      const bigNumberPrice: BigNumber = BigNumber.from(price.toString()).mul(pendingExpo);
      const publishTime = Math.floor(Date.now() / 1000);
      return {
        price: bigNumberPrice,
        expo: -1 * 30,
        publishTime,
      };
    } catch (error) {
      this.logger.error('Error while converting to precision format', error);
    }
  };

  fetchFromBinance = async (): Promise<PriceData> => {
    const prices = {};

    try {
      const symbols = process.env.BINANCE_TOKEN_PAIRS.split(',');
      const symbolsJSON = JSON.stringify(symbols);
      const encodedSymbols = encodeURIComponent(symbolsJSON);
      const binanceTickerUrl = `${process.env.BINANCE_TICKER_URL}?symbols=${encodedSymbols}`;
      const response = await axios.get(binanceTickerUrl); // in binance you can specify most of token pairs prices you need in a single API call (always check API response for your tokenPair before implementing)
      const binancePrices = response.data;

      const kTokens = ['PEPE', 'SHIB', 'BONK'];

      binancePrices.forEach((item) => {
        const symbol = item.symbol.replace(/USDC|USDT/g, ''); // Remove 'USDC' or 'USDT'
        const tokenPrice = parseFloat(item.price);

        if (kTokens.includes(symbol)) {
          prices['k' + symbol] = tokenPrice * 1000;
        }
        prices[symbol] = tokenPrice; // Store the price as a float
      });
    } catch (error) {
      this.logger.error('Error fetching prices from binance', error);
    }

    this.debug ? this.logger.log(`Binance Price: ${JSON.stringify(prices)}`) : '';
    return prices;
  };

  fetchFromCoinbase = async (): Promise<PriceData> => {
    const prices = {};
    try {
      // coinbase has one api to fetch prices of all supported tokens relative to one provided token
      // leverage this get price of USDC, relative to each token filter out prices you need and do 1/tokenPrice
      const currency = 'USDC';
      const coinbaseExchageRateUrl = `${process.env.COINBASE_EXCHANGE_RATE_URL}?currency=${currency}`;
      const response = await axios.get(coinbaseExchageRateUrl); // this does not return BNB and FTM, fetch separately
      const coinbasePrices = response?.data?.data?.rates;
      const coinbaseSupportedToken = process.env.COINBASE_SUPPORTED_TOKENS.split(',');
      const kTokens = ['BONK'];
      coinbaseSupportedToken.forEach((item) => {
        if (coinbasePrices[item] && item != 'USDC') {
          const tokenPrice = 1 / Number(coinbasePrices[item]);

          // Apply 'k' prefix and multiplier for tokens in kTokens list
          if (kTokens.includes(item)) {
            prices['k' + item] = tokenPrice * 1000;
          }

          // Store normalized price without 'k' prefix
          prices[item] = tokenPrice;
        }
      });
    } catch (error) {
      this.logger.error('Error fetching prices from CoinBase', error);
    }

    this.debug ? this.logger.log(`Coinbase Price: ${JSON.stringify(prices)}`) : '';

    return prices;
  };

  fetchFromOkx = async (): Promise<PriceData> => {
    const prices = {};
    const okxMarkPriceUrl = process.env.OKX_PRICEFEED_API;

    try {
      const okxResponse = await axios.get(okxMarkPriceUrl);
      const okxPrices = okxResponse?.data?.data;

      okxPrices.forEach((item) => {
        const [token, pair] = item.instId.split('-');
        if (this.supportedTokens.includes(token) && pair === 'USDC') {
          prices[token] = parseFloat(item.markPx);
        }
      });
    } catch (error) {
      this.logger.error('Error fetching from OKX', error);
    }
    this.debug ? this.logger.log(`OKX Price: ${JSON.stringify(prices)}`) : '';
    return prices;
  };

  fetchFromBybit = async (): Promise<PriceData> => {
    const prices = {};
    const bybitPriceUrl = process.env.BYBIT_PRICEFEED_API;
    this.debug ? this.logger.log(`ByBit Price URL: ${bybitPriceUrl}`) : '';

    // Define lists for specific token handling
    const usdtTokens = ['SCR', 'PONKE', 'WAL', 'FLUID'];
    const kTokens = ['BONK'];

    try {
      const bybitResponse = await axios.get(bybitPriceUrl);
      const bybitPrices = bybitResponse?.data?.result?.list;

      this.supportedTokens.forEach((token) => {
        const tokenPair = token + (usdtTokens.includes(token) ? 'USDT' : 'USDC');

        const found = bybitPrices.find((item) => item.symbol === tokenPair);

        if (kTokens.includes(token)) {
          prices['k' + token] = found ? parseFloat(found.lastPrice) * 1000 : undefined;
        }

        prices[token] = found ? parseFloat(found.lastPrice) : undefined;
      });
    } catch (error) {
      this.logger.error('Error fetching from ByBit', error);

      this.debug ? this.logger.log(`ByBit Price: ${prices}`) : '';
    }
    this.logger.log(`ByBit Prices : ${JSON.stringify(prices)}`);
    return prices;
  };

  fetchFromHtx = async (): Promise<PriceData> => {
    const prices = {};
    const htxPriceUrl = process.env.HTX_PRICEFEED_API;
    // Define lists for tokens with special requirements
    const usdtTokens = ['TON', 'PEPE', 'SHIB', 'GOAT', 'WIF', 'POPCAT', 'FARTCOIN', 'MOODENG', 'BRETT', 'GRASS', 'TAIKO', 'TIA', 'SEI', 'SUI', 'ZRO', 'BOME', 'APE', 'LOGX', 'PONKE', 'SWELL', 'ALGO', 'XLM', 'HBAR', 'MOVE'];
    const kTokens = ['PEPE', 'SHIB', 'BONK'];
    try {
      const htxResponse = await axios.get(htxPriceUrl);
      const htxPrices = htxResponse?.data?.data;

      this.supportedTokens.forEach((token) => {
        // Determine token pair
        const tokenPair = token.toLowerCase() + (usdtTokens.includes(token) ? 'usdt' : 'usdc');

        // Find price data for the token pair
        const found = htxPrices.find((item) => item.symbol === tokenPair);

        // Apply 'k' prefix and multiplier if token is in kTokens list
        if (kTokens.includes(token)) {
          prices['k' + token] = found ? found.close * 1000 : undefined;
        }
        prices[token] = found ? found.close : undefined;
      });
    } catch (error) {
      this.logger.error('Error fetching from HTX', error);
    }
    this.debug ? this.logger.log(`Price fetched from HTX : ${JSON.stringify(prices)}`) : '';
    return prices;
  };

  fetchFromMexc = async (): Promise<PriceData> => {
    const prices = {};
    const mexcPriceUrl = process.env.MEX_PRICEFEED_API;
    const mexcSupportedTokens = process.env.MEX_SUPPORTED_TOKENS;
    try {
      const mexcResponse = await axios.get(mexcPriceUrl);
      const mexcPrices = mexcResponse?.data ?? [];

      // This is not efficient, but we are doing this to keep the code simple
      mexcSupportedTokens.split(',').forEach((tokenPair) => {
        const found = mexcPrices.find((item) => item.symbol === tokenPair);
        // Remove USDT from end
        const tokenSymbol = tokenPair.replace(/USDT$/, '');
        prices[tokenSymbol] = found ? found.price : undefined;
      });
    } catch (error) {
      this.logger.error('Error fetching from MEXC', error);
    }
    this.debug ? this.logger.log(`Price fetched from MEXC : ${JSON.stringify(prices)}`) : '';
    return prices;
  };

  fetchFromGate = async (): Promise<PriceData> => {
    const prices: PriceData = {};
    const gatePriceUrl = process.env.GATE_TICKER_API;
    const gateSupportedTokens = process.env.GATE_SUPPORTED_TOKEN;

    try {
      const gateResponse = await axios.get(gatePriceUrl);
      const gatePrices = gateResponse?.data ?? [];

      // Process each token in the supported tokens list
      gateSupportedTokens.split(',').forEach((tokenPair) => {
        const found = gatePrices.find((item) => item.currency_pair === tokenPair);

        // Remove "_USDC" or "_USDT" from the tokenPair name
        const tokenSymbol = tokenPair.replace(/_(USDC|USDT)$/, '');
        prices[tokenSymbol] = found ? found.last : undefined;
      });
    } catch (error) {
      console.error('Error fetching from Gate API', error);
      this.debug ? this.logger.log(`Price fetched from GATE : ${JSON.stringify(prices)}`) : '';
    }

    console.log(`Price fetched from Gate API: ${JSON.stringify(prices)}`);
    return prices;
  };

  fetchFromKraken = async (): Promise<PriceData> => {
    const prices = {};
    const krakenPriceUrl = process.env.KRAKEN_PRICEFEED_API;

    try {
      const krakenResponse = await axios.get(krakenPriceUrl);
      const krakenPrices = krakenResponse?.data?.result;

      //Note we are currently hardcoding the keys in the price dict since kraken prices do not follow a codable pattern
      prices['BTC'] = krakenPrices['WBTCUSD'] ? parseFloat(krakenPrices['WBTCUSD'].c[0]) : undefined;
      prices['ETH'] = krakenPrices['ETHUSDC'] ? parseFloat(krakenPrices['ETHUSDC'].c[0]) : undefined;
      prices['USDC'] = krakenPrices['USDCUSD'] ? parseFloat(krakenPrices['USDCUSD'].c[0]) : undefined;
      prices['PEPE'] = krakenPrices['PEPEUSD'] ? parseFloat(krakenPrices['PEPEUSD'].c[0]) : undefined;
      prices['TON'] = krakenPrices['TONUSD'] ? parseFloat(krakenPrices['TONUSD'].c[0]) : undefined;
      prices['SOL'] = krakenPrices['SOLUSD'] ? parseFloat(krakenPrices['SOLUSD'].c[0]) : undefined;
      prices['BTC'] = prices['BTC'] / prices['USDC'];
      prices['PEPE'] = krakenPrices['PEPEUSD'] ? parseFloat(krakenPrices['PEPEUSD'].c[0]) : undefined;
      prices['kPEPE'] = krakenPrices['PEPEUSD'] ? parseFloat(krakenPrices['PEPEUSD'].c[0]) * 1000 : undefined;
      prices['SHIB'] = krakenPrices['SHIBUSD'] ? parseFloat(krakenPrices['SHIBUSD'].c[0]) : undefined;
      prices['kSHIB'] = krakenPrices['SHIBUSD'] ? parseFloat(krakenPrices['SHIBUSD'].c[0]) * 1000 : undefined;
      prices['BONK'] = krakenPrices['BONKUSD'] ? parseFloat(krakenPrices['BONKUSD'].c[0]) : undefined;
      prices['kBONK'] = krakenPrices['BONKUSD'] ? parseFloat(krakenPrices['BONKUSD'].c[0]) * 1000 : undefined;
      prices['SEI'] = krakenPrices['SEIUSD'] ? parseFloat(krakenPrices['SEIUSD'].c[0]) : undefined;
      prices['SUI'] = krakenPrices['SUIUSD'] ? parseFloat(krakenPrices['SUIUSD'].c[0]) : undefined;
      prices['ZRO'] = krakenPrices['ZROUSD'] ? parseFloat(krakenPrices['ZROUSD'].c[0]) : undefined;
      prices['TIA'] = krakenPrices['TIAUSD'] ? parseFloat(krakenPrices['TIAUSD'].c[0]) : undefined;
      prices['PONKE'] = krakenPrices['PONKEUSD'] ? parseFloat(krakenPrices['PONKEUSD'].c[0]) : undefined;
      prices['SWELL'] = krakenPrices['SWELLUSD'] ? parseFloat(krakenPrices['SWELLUSD'].c[0]) : undefined;
      prices['WAL'] = krakenPrices['WALUSD'] ? parseFloat(krakenPrices['WALUSD'].c[0]) / prices['USDC'] : undefined;

      prices['SOL'] = prices['SOL'] / prices['USDC'];
    } catch (error) {
      this.logger.error('Error fetching from Kraken', error);
    }
    this.debug ? this.logger.log(`Kraken Prices: ${JSON.stringify(prices)}`) : '';
    return prices;
  };

  fetchFromRedstone = async (): Promise<PriceData> => {
    const prices = {};
    const redstoneSupportedTokens = process.env.REDSTONE_SUPPORTED_TOKENS;
    const redstoneUrl = process.env.REDSTONE_GENERIC_API + redstoneSupportedTokens + '&provider=redstone';

    try {
      const redstoneResponse = await axios.get(redstoneUrl);
      const kTokens = ['PEPE', 'SHIB', 'BONK'];

      //get usdc price first
      prices['USDC'] = redstoneResponse.data['USDC'].value;

      redstoneSupportedTokens.split(',').forEach((token) => {
        const priceData = redstoneResponse.data;
        if (priceData.hasOwnProperty(token)) {
          if (token == 'USDC') {
            return;
          }
          // Apply 'k' prefix and multiplier for specific tokens
          const normalizedPrice = priceData[token].value / prices['USDC'];

          if (kTokens.includes(token)) {
            prices['k' + token] = normalizedPrice * 1000;
          }
          prices[token] = normalizedPrice;
        }
      });
    } catch (error) {
      this.logger.error('Error fetching from Redstone', error);
    }
    this.debug ? this.logger.log(`Redstone Prices: ${JSON.stringify(prices)}`) : '';
    return prices;
  };

  fetchFromPolyMarket = async (): Promise<any> => {
    const prices = [];
    const polyMarketUrl = process.env.POLYMARKET_PRICE_API;
    const tokenIds = (process.env.POLYMARKET_TOKEN_IDS || '').split(',').filter(Boolean);
    const tokenNames = (process.env.POLYMARKET_TOKEN_NAMES || '').split(',').filter(Boolean);

    for (let i = 0; i < tokenIds.length; i++) {
      try {
        const response = await axios.get(polyMarketUrl + tokenIds[i] + '&side=buy');
        prices.push(response.data);
      } catch (error) {
        this.logger.error('Error fetching from PolyMarket', error);
      }
      this.debug ? this.logger.log(`${tokenNames[i]} PolyMarket Price: ${JSON.stringify(prices[i])}`) : '';
    }
    return prices;
  };

  getNextFinnhubApiKey(): string {
    const key = this.finnhubApiKeys[this.currentKeyIndex];
    this.currentKeyIndex = (this.currentKeyIndex + 1) % this.finnhubApiKeys.length;
    return key;
  }

  getNextAllticksApiKey(): string {
    const key = this.allticksApiKeys[this.currentAllticksKeyIndex];
    this.currentAllticksKeyIndex = (this.currentAllticksKeyIndex + 1) % this.allticksApiKeys.length;
    return key;
  }

  fetchFromFinnhub = async (): Promise<PriceData> => {
    const prices = {};

    if (this.cachedStockPrices) {
      Object.entries(this.cachedStockPrices).forEach(([stock, data]) => {
        prices[stock] = data.price;

        if (this.debug) {
          const ageInSeconds = Math.floor(Date.now() / 1000) - data.timestamp;
          this.logger.log(`Using cached price for ${stock}, age: ${ageInSeconds}s`);
        }
      });
    }

    this.debug ? this.logger.log(`Finnhub Cached Prices: ${JSON.stringify(prices)}`) : '';
    return prices;
  };

  fetchFromAllticks = async (): Promise<PriceData> => {
    const prices = {};

    if (this.cachedAllticksStockPrices) {
      Object.entries(this.cachedAllticksStockPrices).forEach(([stock, data]) => {
        prices[stock] = data.price;

        if (this.debug) {
          const ageInSeconds = Math.floor(Date.now() / 1000) - data.timestamp;
          this.logger.log(`Using cached Allticks price for ${stock}, age: ${ageInSeconds}s`);
        }
      });
    }

    if (this.cachedAllticksCommodityPrices) {
      Object.entries(this.cachedAllticksCommodityPrices).forEach(([commodity, data]) => {
        prices[commodity] = data.price;

        if (this.debug) {
          const ageInSeconds = Math.floor(Date.now() / 1000) - data.timestamp;
          this.logger.log(`Using cached Allticks commodity price for ${commodity}, age: ${ageInSeconds}s`);
        }
      });
    }

    this.debug ? this.logger.log(`Allticks Cached Prices: ${JSON.stringify(prices)}`) : '';
    return prices;
  };

  fetchFromTiingo = async (): Promise<PriceData> => {
    const prices = {};

    if (this.cachedTiingoStockPrices) {
      Object.entries(this.cachedTiingoStockPrices).forEach(([stock, data]) => {
        prices[stock] = data.price;

        if (this.debug) {
          const ageInSeconds = Math.floor(Date.now() / 1000) - data.timestamp;
          this.logger.log(`Using cached Tiingo price for ${stock}, age: ${ageInSeconds}s`);
        }
      });
    }

    this.debug ? this.logger.log(`Tiingo Cached Prices: ${JSON.stringify(prices)}`) : '';
    return prices;
  };

  fetchFromTiingoForex = async (): Promise<PriceData> => {
    const prices = {};

    if (this.cachedTiingoForexPrices) {
      Object.entries(this.cachedTiingoForexPrices).forEach(([normalizedSymbol, data]) => {
        prices[normalizedSymbol] = data.price;

        if (this.debug) {
          const ageInSeconds = Math.floor(Date.now() / 1000) - data.timestamp;
          this.logger.log(`Using cached Tiingo Forex price for ${normalizedSymbol}, age: ${ageInSeconds}s`);
        }
      });
    }

    this.debug ? this.logger.log(`Tiingo Forex Cached Prices: ${JSON.stringify(prices)}`) : '';
    return prices;
  };

  fetchFromHyperliquid = async (): Promise<PriceData> => {
    const prices = {};
    const hyperliquidUrl = 'https://api.hyperliquid.xyz/info';

    // Group tokens by dex prefix (e.g., "xyz:AAPL" -> dex: "xyz", token: "xyz:AAPL")
    // No prefix means main Hyperliquid dex (empty string)
    const tokensByDex: Record<string, string[]> = {};

    this.hyperliquidTokens.forEach((token) => {
      const colonIndex = token.indexOf(':');
      const dex = colonIndex > 0 ? token.substring(0, colonIndex) : '';
      if (!tokensByDex[dex]) {
        tokensByDex[dex] = [];
      }
      tokensByDex[dex].push(token);
    });

    try {
      // Make parallel requests for each dex
      const dexes = Object.keys(tokensByDex);
      const requests = dexes.map((dex) => {
        const payload: { type: string; dex?: string } = { type: 'allMids' };
        if (dex) {
          payload.dex = dex;
        }
        return axios.post(hyperliquidUrl, payload);
      });

      const responses = await Promise.all(requests);

      // Process responses for each dex
      responses.forEach((response, index) => {
        const dex = dexes[index];
        const tokensForDex = tokensByDex[dex];

        if (response.data) {
          tokensForDex.forEach((token) => {
            const price = response.data[token];
            if (price && parseFloat(price) > 0) {
              // Map Hyperliquid token names to internal names (e.g., xyz:XYZ100 -> XYZ100)
              const mappedToken = HYPERLIQUID_TOKEN_MAP[token] || token;
              prices[mappedToken] = parseFloat(price);
            }
          });
        }
      });
    } catch (error) {
      this.logger.error('Error fetching from Hyperliquid', error);
    }

    this.debug ? this.logger.log(`Hyperliquid Prices: ${JSON.stringify(prices)}`) : '';
    return prices;
  };

  getPriceForSupportedTokens = async () => {
    const currentTimestamp = new Date().getTime();

    const exchangesRates = [];
    const polyMarketPricesFormatted = [];
    let usdcPriceFormatted;

    try {
      const [binancePrices, coinbasePrices, OkxPrices, BybitPrices, KrakenPrices, HtxPrices, redstonePrices, cantonPrices, polyMarketPrices, mexcPrices, gatePrices, finnhubPrices, allticksPrices, tiingoPrices, tiingoForexPrices, hyperliquidPrices] = await Promise.all([this.fetchFromBinance(), this.fetchFromCoinbase(), this.fetchFromOkx(), this.fetchFromBybit(), this.fetchFromKraken(), this.fetchFromHtx(), this.fetchFromRedstone(), this.fetchFromCanton(), this.fetchFromPolyMarket(), this.fetchFromMexc(), this.fetchFromGate(), this.fetchFromFinnhub(), this.fetchFromAllticks(), this.fetchFromTiingo(), this.fetchFromTiingoForex(), this.fetchFromHyperliquid()]);

      const exchangeVariables = {
        RedStonePricePill: cantonPrices,
        Binance: binancePrices,
        Coinbase: coinbasePrices,
        OKX: OkxPrices,
        Bybit: BybitPrices,
        Kraken: KrakenPrices,
        Htx: HtxPrices,
        RedStone: redstonePrices,
        Mexc: mexcPrices,
        Gate: gatePrices,
        Finnhub: finnhubPrices,
        Allticks: allticksPrices,
        Tiingo: tiingoPrices,
        TiingoForex: tiingoForexPrices,
        Hyperliquid: hyperliquidPrices,
      };

      for (const exchange in this.exchangeWeights) {
        if (exchange === 'RedStonePricePill') continue; // Canton handled via primary path
        const weight = this.exchangeWeights[exchange];
        const exchangeVariable = exchangeVariables[exchange];
        for (let i = 0; i < weight; i++) {
          exchangesRates.push(exchangeVariable);
        }
      }
      for (let i = 0; i < polyMarketPrices.length; i++) {
        polyMarketPricesFormatted.push(this.convertToPrecisionFormat(polyMarketPrices[i].price));
      }
      // usdc has a fixed price of 1
      usdcPriceFormatted = this.convertToPrecisionFormat(1);

      // Order of supported Tokens matter
      this.allSupportedTokens.forEach((token) => {
        if (token === 'USDC' || token === 'apeUSD') {
          return;
        } // skip USDC and apeUSD

        // === PRIMARY PATH: Canton PricePill tokens ===
        if (this.cantonPrimaryTokens.has(token)) {
          const cantonPrice = cantonPrices?.[token];
          const isStale = this.cantonPricePillService.isFeedStale(token);

          if (cantonPrice && cantonPrice > 0 && !isStale) {
            const precisionPrice = this.convertToPrecisionFormat(cantonPrice);
            this.sendToRedis(precisionPrice, token);
            this.debug ? this.logger.log(`[PRIMARY] ${token} price from Canton PricePill: ${cantonPrice}`) : '';
            return;
          }

          // === SECONDARY PATH: Fallback to weighted median excluding RedStonePricePill ===
          if (isStale && cantonPrice && cantonPrice > 0) {
            this.logger.warn(`[FALLBACK] ${token}: Canton PricePill data is STALE, falling back to weighted median`);
          } else {
            this.logger.warn(`[FALLBACK] ${token}: Canton PricePill price unavailable, falling back to weighted median`);
          }

          const fallbackPrices: number[] = [];
          for (const exchange in this.exchangeWeights) {
            if (exchange === 'RedStonePricePill') continue; // Exclude Canton from fallback
            const weight = this.exchangeWeights[exchange];
            const exchangeData = exchangeVariables[exchange];
            if (exchangeData && exchangeData[token]) {
              for (let i = 0; i < weight; i++) {
                fallbackPrices.push(exchangeData[token]);
              }
            }
          }

          const fallbackMedian = this.findMedian(fallbackPrices);
          if (fallbackMedian) {
            const precisionPrice = this.convertToPrecisionFormat(fallbackMedian);
            this.sendToRedis(precisionPrice, token);
            this.logger.warn(`[FALLBACK] ${token} secondary median price: ${fallbackMedian}`);
          } else {
            this.logger.error(`ALERT!! No price available for ${token} from any source at ${Math.floor(Date.now() / 1000)}`, '');
          }
          return;
        }

        // === STANDARD PATH: Non-Canton tokens, unchanged ===
        const tokenPrices = [];
        exchangesRates.forEach((rates) => {
          if (rates && rates[token]) {
            tokenPrices.push(rates[token]);
          }
        });

        const medianPrice = this.findMedian(tokenPrices);
        if (medianPrice) {
          const precisionPrice = this.convertToPrecisionFormat(medianPrice);
          this.sendToRedis(precisionPrice, token);
        } else {
          this.logger.log(`ALERT!! Price does not exist for ${token} at ${Math.floor(Date.now() / 1000)}, skipped writting to redis`);
        }
      });

      // Push PolyMarket prices to redis
      for (let i = 0; i < polyMarketPricesFormatted.length; i++) {
        await this.sendToRedis(polyMarketPricesFormatted[i], process.env.POLYMARKET_TOKEN_NAMES.split(',')[i]);
      }

      await this.sendToRedis(usdcPriceFormatted, 'USDC');
      await this.sendToRedis(usdcPriceFormatted, 'apeUSD');
    } catch (error) {
      this.logger.error('Error fetching price from any of the exchange', error);
    }

    this.debug ? this.logger.log(`time taken ${new Date().getTime() - currentTimestamp}`) : '';
  };

  startAfterCertainTime = (functionName, timeOut: number, frequency: number) => {
    setTimeout(() => {
      setInterval(functionName, frequency);
    }, timeOut);
  };

  fetchFinnhubStocks = async () => {
    const stocks = this.parseEnvList(process.env.FINNHUB_STOCKS);

    if (stocks.length === 0 || this.finnhubApiKeys.length === 0) {
      return;
    }

    this.logger.log(`Starting Finnhub fetch for ${stocks.length} stocks`);
    const fetchResults: Record<string, number> = {};
    const failedStocks: string[] = [];

    const fetchPromises = stocks.map(async (stock) => {
      const apiKey = this.getNextFinnhubApiKey();

      try {
        const finnhubUrl = `${process.env.FINNHUB_API_URL}?symbol=${stock}&token=${apiKey}`;
        const response = await axios.get(finnhubUrl);

        if (response.data && response.data.c) {
          const stockPrice = parseFloat(response.data.c);

          this.cachedStockPrices[stock] = {
            price: stockPrice,
            timestamp: Math.floor(Date.now() / 1000),
          };

          const precisionPrice = this.convertToPrecisionFormat(stockPrice);
          await this.sendToRedis(precisionPrice, stock);

          fetchResults[stock] = stockPrice;
          return true;
        } else {
          failedStocks.push(stock);
          return false;
        }
      } catch (error) {
        failedStocks.push(stock);
        this.logger.error(`Error fetching ${stock} from Finnhub:`, error.stack);
        return false;
      }
    });

    try {
      await Promise.all(fetchPromises);

      if (Object.keys(fetchResults).length > 0) {
        this.logger.log(`Finnhub prices fetched successfully: ${JSON.stringify(fetchResults)}`);
      }

      if (failedStocks.length > 0) {
        this.logger.warn(`Failed to fetch prices for stocks: ${failedStocks.join(', ')}`);
      }

      this.logger.log(`Completed Finnhub fetch cycle: ${Object.keys(fetchResults).length} succeeded, ${failedStocks.length} failed`);
    } catch (error) {
      this.logger.error('Error in Finnhub batch processing', error.stack);
    }
  };

  fetchTiingoForex = async () => {
    const forexPairs = this.parseEnvList(process.env.TIINGO_FOREX_PAIRS);

    if (forexPairs.length === 0 || !this.tiingoApiToken) {
      return;
    }

    const fetchResults: Record<string, number> = {};
    const failedPairs: string[] = [];

    try {
      const tickersString = forexPairs.join(',');
      const tiingoForexUrl = `https://api.tiingo.com/tiingo/fx/top?tickers=${tickersString}&token=${this.tiingoApiToken}`;

      this.logger.log(`Fetching Tiingo Forex data for pairs: ${tickersString}`);

      const response = await axios.get(tiingoForexUrl);

      if (response.data && Array.isArray(response.data)) {
        response.data.forEach((forexData: any) => {
          try {
            const originalTicker = forexData.ticker?.toUpperCase();
            const midPrice = forexData.midPrice;

            if (midPrice && originalTicker) {
              // Normalize forex pair to base currency (e.g., EURUSD -> EUR, AUDUSD -> AUD)
              const normalizedSymbol = normalizeForexSymbol(originalTicker);

              this.cachedTiingoForexPrices[normalizedSymbol] = {
                price: midPrice,
                timestamp: Math.floor(Date.now() / 1000),
              };

              const precisionPrice = this.convertToPrecisionFormat(midPrice);
              this.sendToRedis(precisionPrice, normalizedSymbol);

              fetchResults[normalizedSymbol] = midPrice;

              this.logger.log(`Normalized ${originalTicker} -> ${normalizedSymbol}: ${midPrice}`);
            } else {
              failedPairs.push(originalTicker || 'unknown');
            }
          } catch (error) {
            const ticker = forexData?.ticker || 'unknown';
            failedPairs.push(ticker);
            this.logger.error(`Error processing ${ticker} from Tiingo Forex response:`, error.stack);
          }
        });
      } else {
        this.logger.warn('Invalid response format from Tiingo Forex API');
        failedPairs.push(...forexPairs);
      }

      if (Object.keys(fetchResults).length > 0) {
        this.logger.log(`Tiingo Forex prices fetched successfully: ${JSON.stringify(fetchResults)}`);
      }

      if (failedPairs.length > 0) {
        this.logger.warn(`Failed to fetch prices for forex pairs from Tiingo: ${failedPairs.join(', ')}`);
      }

      this.logger.log(`Completed Tiingo Forex fetch cycle: ${Object.keys(fetchResults).length} succeeded, ${failedPairs.length} failed`);
    } catch (error) {
      this.logger.error('Error in Tiingo Forex batch processing', error.stack);
      failedPairs.push(...forexPairs);
    }
  };

  fetchTiingoStocks = async () => {
    const stocks = this.parseEnvList(process.env.TIINGO_STOCKS);

    if (stocks.length === 0 || !this.tiingoApiToken) {
      return;
    }

    const fetchResults: Record<string, number> = {};
    const failedStocks: string[] = [];

    try {
      const tickersString = stocks.join(',');
      const tiingoUrl = `https://api.tiingo.com/iex/?tickers=${tickersString}&token=${this.tiingoApiToken}`;

      const response = await axios.get(tiingoUrl);

      if (response.data && Array.isArray(response.data)) {
        response.data.forEach((stockData: any) => {
          try {
            let ticker = stockData.ticker;
            if (ticker === 'BRK-B') { ticker = 'BRK.B'; }

            const stockPrice = stockData.tngoLast || stockData.last;

            if (stockPrice && ticker) {
              this.cachedTiingoStockPrices[ticker] = {
                price: stockPrice,
                timestamp: Math.floor(Date.now() / 1000),
              };

              const precisionPrice = this.convertToPrecisionFormat(stockPrice);
              this.sendToRedis(precisionPrice, ticker);

              fetchResults[ticker] = stockPrice;
            } else {
              failedStocks.push(ticker || 'unknown');
            }
          } catch (error) {
            const ticker = stockData?.ticker || 'unknown';
            failedStocks.push(ticker);
            this.logger.error(`Error processing ${ticker} from Tiingo response:`, error.stack);
          }
        });
      } else {
        this.logger.warn('Invalid response format from Tiingo API');
        failedStocks.push(...stocks);
      }

      if (Object.keys(fetchResults).length > 0) {
        this.logger.log(`Tiingo prices fetched successfully: ${JSON.stringify(fetchResults)}`);
      }

      if (failedStocks.length > 0) {
        this.logger.warn(`Failed to fetch prices for stocks from Tiingo: ${failedStocks.join(', ')}`);
      }

      this.logger.log(`Completed Tiingo fetch cycle: ${Object.keys(fetchResults).length} succeeded, ${failedStocks.length} failed`);
    } catch (error) {
      this.logger.error('Error in Tiingo batch processing', error.stack);
      failedStocks.push(...stocks);
    }
  };

  fetchAllticksStocks = async () => {
    const stocks = this.parseEnvList(process.env.ALLTICKS_STOCKS);

    if (stocks.length === 0 || this.allticksApiKeys.length === 0) {
      return;
    }

    this.logger.log(`Starting Allticks fetch for ${stocks.length} stocks`);
    const fetchResults: Record<string, number> = {};
    const failedStocks: string[] = [];

    // Batch stocks (max 5 per batch for free tier)
    const batchSize = Number(process.env.ALLTICKS_BATCH_SIZE || 5);
    const stockBatches = [];

    for (let i = 0; i < stocks.length; i += batchSize) {
      stockBatches.push(stocks.slice(i, i + batchSize));
    }

    // Process batches with API key rotation
    for (let batchIndex = 0; batchIndex < stockBatches.length; batchIndex++) {
      const batch = stockBatches[batchIndex];
      const apiKey = this.getNextAllticksApiKey();

      try {
        // Generate unique trace ID
        const traceId = `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

        // Prepare request body for batch
        const requestBody = {
          trace: traceId,
          data: {
            symbol_list: batch.map((stock) => ({ code: stock })),
          },
        };

        // URL encode the query data
        const queryData = encodeURIComponent(JSON.stringify(requestBody));
        const allticksUrl = `https://quote.alltick.io/quote-stock-b-api/trade-tick?token=${apiKey}&query=${queryData}`;

        this.logger.log(`Fetching Allticks batch: ${batch.join(', ')} with API key: ${apiKey.substring(0, 8)}...`);

        const response = await axios.get(allticksUrl, { timeout: 8000 });

        if (response.data && response.data.ret === 200 && response.data.data && response.data.data.tick_list) {
          const ticks = response.data.data.tick_list;

          ticks.forEach(async (tick: any) => {
            if (tick.price && parseFloat(tick.price) > 0) {
              const stockPrice = parseFloat(tick.price);

              // Extract stock symbol from code (remove .HK, .US, etc.) Allticks api has this format
              let stockSymbol = tick.code.split('.')[0];
              const stockExchange = tick.code.split('.')[1];

              // For HK stocks, use ticker name if available, otherwise use original format
              if (stockExchange === 'HK') {
                const ticker = NUMERICAL_STOCK_TICKER_MAP[tick.code];
                if (ticker) {
                  stockSymbol = ticker;
                } else {
                  stockSymbol = tick.code; // Use full format like "1024.HK" if no mapping
                }
              }

              this.cachedAllticksStockPrices[stockSymbol] = {
                price: stockPrice,
                timestamp: Math.floor(Date.now() / 1000),
              };

              fetchResults[stockSymbol] = stockPrice;

              // Send to Redis with precision format
              const precisionPrice = this.convertToPrecisionFormat(stockPrice);
              await this.sendToRedis(precisionPrice, stockSymbol);
            }
          });

          this.logger.log(`Successfully processed Allticks batch with ${ticks.length} ticks`);
        } else {
          this.logger.warn(`Invalid Allticks response for batch: ${batch.join(', ')}`);
          failedStocks.push(...batch);
        }
      } catch (error) {
        this.logger.error(`Error fetching Allticks batch [${batch.join(', ')}]:`, error.stack);
        failedStocks.push(...batch);
      }

      // Add delay between batches to respect rate limits
      if (batchIndex < stockBatches.length - 1) {
        await new Promise((resolve) => setTimeout(resolve, 1000)); // 1 second delay between batches
      }
    }

    if (Object.keys(fetchResults).length > 0) {
      this.logger.log(`Allticks prices fetched successfully: ${JSON.stringify(fetchResults)}`);
    }

    if (failedStocks.length > 0) {
      this.logger.warn(`Failed to fetch Allticks prices for stocks: ${failedStocks.join(', ')}`);
    }

    this.logger.log(`Completed Allticks fetch cycle: ${Object.keys(fetchResults).length} succeeded, ${failedStocks.length} failed`);
  };

  fetchAllticksCommodities = async () => {
    const commodities = this.parseEnvList(process.env.ALLTICKS_COMMODITIES);

    if (commodities.length === 0 || this.allticksApiKeys.length === 0) {
      return;
    }

    this.logger.log(`Starting Allticks commodities fetch for ${commodities.length} commodities`);
    const fetchResults: Record<string, number> = {};
    const failedCommodities: string[] = [];

    // Batch commodities (max 5 per batch for free tier)
    const batchSize = Number(process.env.ALLTICKS_BATCH_SIZE || 5);
    const commodityBatches = [];

    for (let i = 0; i < commodities.length; i += batchSize) {
      commodityBatches.push(commodities.slice(i, i + batchSize));
    }

    // Process batches with API key rotation
    for (let batchIndex = 0; batchIndex < commodityBatches.length; batchIndex++) {
      const batch = commodityBatches[batchIndex];
      const apiKey = this.getNextAllticksApiKey();

      try {
        // Generate unique trace ID
        const traceId = `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

        // Prepare request body for batch
        const requestBody = {
          trace: traceId,
          data: {
            symbol_list: batch.map((commodity) => ({ code: commodity })),
          },
        };

        // URL encode the query data
        const queryData = encodeURIComponent(JSON.stringify(requestBody));
        const allticksUrl = `https://quote.alltick.io/quote-b-api/trade-tick?token=${apiKey}&query=${queryData}`;

        this.logger.log(`Fetching Allticks commodities batch: ${batch.join(', ')} with API key: ${apiKey.substring(0, 8)}...`);

        const response = await axios.get(allticksUrl, { timeout: 8000 });

        if (response.data && response.data.ret === 200 && response.data.data && response.data.data.tick_list) {
          const ticks = response.data.data.tick_list;

          ticks.forEach(async (tick: any) => {
            if (tick.price && parseFloat(tick.price) > 0) {
              const commodityPrice = parseFloat(tick.price);
              const commoditySymbol = tick.code;

              this.cachedAllticksCommodityPrices[commoditySymbol] = {
                price: commodityPrice,
                timestamp: Math.floor(Date.now() / 1000),
              };

              fetchResults[commoditySymbol] = commodityPrice;

              // Send to Redis with precision format
              const precisionPrice = this.convertToPrecisionFormat(commodityPrice);
              await this.sendToRedis(precisionPrice, commoditySymbol);
            }
          });

          this.logger.log(`Successfully processed Allticks commodities batch with ${ticks.length} ticks`);
        } else {
          this.logger.warn(`Invalid Allticks commodities response for batch: ${batch.join(', ')}`);
          failedCommodities.push(...batch);
        }
      } catch (error) {
        this.logger.error(`Error fetching Allticks commodities batch [${batch.join(', ')}]:`, error.stack);
        failedCommodities.push(...batch);
      }

      // Add delay between batches to respect rate limits
      if (batchIndex < commodityBatches.length - 1) {
        await new Promise((resolve) => setTimeout(resolve, 1000)); // 1 second delay between batches
      }
    }

    if (Object.keys(fetchResults).length > 0) {
      this.logger.log(`Allticks commodities prices fetched successfully: ${JSON.stringify(fetchResults)}`);
    }

    if (failedCommodities.length > 0) {
      this.logger.warn(`Failed to fetch Allticks commodities prices for: ${failedCommodities.join(', ')}`);
    }

    this.logger.log(`Completed Allticks commodities fetch cycle: ${Object.keys(fetchResults).length} succeeded, ${failedCommodities.length} failed`);
  };

  private parseEnvList(envVar: string | undefined): string[] {
    return (
      envVar
        ?.split(',')
        .map((item) => item.trim())
        .filter((item) => item !== '') || []
    );
  }
}
