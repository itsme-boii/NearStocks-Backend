import { Controller, Get, Query, ValidationPipe, BadRequestException, InternalServerErrorException } from '@nestjs/common';
import { AppService } from './app.service';
import { WinstonLogger } from './utils/winston.service';
import { NUMERICAL_STOCK_TICKER_MAP } from './config/numericalStockTicker.config';
import { parseAndNormalizeForexPairs } from './utils/forex.utils';

@Controller()
export class AppController {
  private supportedTokens: string[];
  private logger: WinstonLogger;

  constructor(private readonly appService: AppService) {
    this.logger = new WinstonLogger('AppController');
    const tokens: string = process.env.SUPPORTED_TOKENS || '';
    this.supportedTokens = tokens
      .split(',')
      .map((token) => token.trim())
      .filter((token) => token !== '');

    const finnhubStocks =
      process.env.FINNHUB_STOCKS?.split(',')
        .map((token) => token.trim())
        .filter((token) => token !== '') || [];
    this.supportedTokens.push(...finnhubStocks);

    const tiingoStocks =
      process.env.TIINGO_STOCKS?.split(',')
        .map((token) => token.trim())
        .filter((token) => token !== '') || [];
    this.supportedTokens.push(...tiingoStocks);

    // Parse and normalize forex pairs from environment variable
    const normalizedForexPairs = parseAndNormalizeForexPairs(process.env.TIINGO_FOREX_PAIRS);
    this.supportedTokens.push(...normalizedForexPairs);

    // TODO: Move this to env in ALL_SUPPORTED_TOKENS and SUPPORTED_TOKENS
    const allticksStocks =
      process.env.ALLTICKS_STOCKS?.split(',')
        .map((token) => token.trim())
        .filter((token) => token !== '') || [];
    const transformedAllticksStocks = allticksStocks.map((stock) => {
      return NUMERICAL_STOCK_TICKER_MAP[stock] || stock;
    });
    this.supportedTokens.push(...transformedAllticksStocks);

    // TODO: Move this to env in ALL_SUPPORTED_TOKENS and SUPPORTED_TOKENS
    const allticksCommodities =
      process.env.ALLTICKS_COMMODITIES?.split(',')
        .map((token) => token.trim())
        .filter((token) => token !== '') || [];
    this.supportedTokens.push(...allticksCommodities);

    const polyMarketTokens = process.env.POLYMARKET_TOKEN_NAMES?.split(',') || [];
    this.supportedTokens.push(...polyMarketTokens.map((token) => token.trim()).filter((token) => token !== ''));

    this.supportedTokens = [...new Set(this.supportedTokens)];
  }

  private isValidToken(token: string): boolean {
    return this.supportedTokens.includes(token);
  }

  @Get('prices')
  async getPrices(@Query('tokens', ValidationPipe) tokens: string) {
    this.logger.log(`Received request for tokens: ${tokens}`);

    try {
      if (!tokens) {
        throw new BadRequestException('Query parameter "tokens" is required');
      }

      const tokenList = tokens.split(',').map((token) => token.trim());

      const invalidTokens = tokenList.filter((token) => !this.isValidToken(token));
      if (invalidTokens.length > 0) {
        throw new BadRequestException(`Unsupported tokens: ${invalidTokens.join(', ')}`);
      }

      const prices = await this.appService.getMultipleDarkOraclePrices(tokenList);
      const returnObj = {
        data: prices,
        tokens: tokenList,
        message: 'Success',
      };

      return returnObj;
    } catch (error) {
      if (error instanceof BadRequestException) {
        this.logger.error(`BadRequestException: ${error.message}`, error.stack);
        throw error;
      } else {
        this.logger.error('Server Error:', error.stack);
        throw new InternalServerErrorException('Server Error');
      }
    }
  }

  @Get('allprices')
  async getAllPrices() {
    try {
      const prices = await this.appService.getMultipleDarkOraclePrices(this.supportedTokens);
      const returnObj = {
        data: prices,
        tokens: this.supportedTokens,
        message: 'Success',
      };

      return returnObj;
    } catch (error) {
      this.logger.error('Server Error:', error.stack);
      throw new InternalServerErrorException('Server Error');
    }
  }

  @Get('health')
  async validate() {
    return {
      status: 'ok',
      message: 'Service is running',
      timestamp: new Date().toISOString(),
    };
  }
}
