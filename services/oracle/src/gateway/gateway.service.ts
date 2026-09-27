import { WebSocketGateway, WebSocketServer, SubscribeMessage, OnGatewayConnection, OnGatewayDisconnect, OnGatewayInit, MessageBody, ConnectedSocket } from '@nestjs/websockets';
import { Server, Socket } from 'socket.io';
import { NUMERICAL_STOCK_TICKER_MAP } from 'src/config/numericalStockTicker.config';
import { RedisService } from 'src/redis/redis.service';
import { parseAndNormalizeForexPairs } from 'src/utils/forex.utils';

@WebSocketGateway()
export class GatewayService implements OnGatewayConnection, OnGatewayDisconnect, OnGatewayInit {
  @WebSocketServer() server: Server;

  constructor(private redisService: RedisService) {}
  async afterInit() {
    await this.redisService.onModuleInit();
  }

  // fetch all the tokne together.
  async setupRedisSubscriptions(client: Socket) {
    let supportedTokens = process.env.ALL_SUPPORTED_TOKENS?.split(',').map((s) => s.trim()).filter((s) => s !== '') || [];
    const polyMarketTokens = process.env.POLYMARKET_TOKEN_NAMES?.split(',').map((s) => s.trim()).filter((s) => s !== '') || [];
    supportedTokens.push(...polyMarketTokens);
    const finnhubStocks =
      process.env.FINNHUB_STOCKS?.split(',')
        .map((s) => s.trim())
        .filter((s) => s !== '') || [];
    supportedTokens.push(...finnhubStocks);

    const tiingoStocks =
      process.env.TIINGO_STOCKS?.split(',')
        .map((s) => s.trim())
        .filter((s) => s !== '') || [];
    supportedTokens.push(...tiingoStocks);

    // Parse and normalize forex pairs from environment variable
    const normalizedForexPairs = parseAndNormalizeForexPairs(process.env.TIINGO_FOREX_PAIRS);
    supportedTokens.push(...normalizedForexPairs);

    // TODO: Move this to env in ALL_SUPPORTED_TOKENS and SUPPORTED_TOKENS
    const allticksStocks =
      process.env.ALLTICKS_STOCKS?.split(',')
        .map((s) => s.trim())
        .filter((s) => s !== '') || [];

    const transformedAllticksStocks = allticksStocks.map((stock) => {
      return NUMERICAL_STOCK_TICKER_MAP[stock] || stock;
    });
    supportedTokens.push(...transformedAllticksStocks);

    // TODO: Move this to env in ALL_SUPPORTED_TOKENS and SUPPORTED_TOKENS
    const allticksCommodities =
      process.env.ALLTICKS_COMMODITIES?.split(',')
        .map((s) => s.trim())
        .filter((s) => s !== '') || [];
    supportedTokens.push(...allticksCommodities);

    supportedTokens = [...new Set(supportedTokens)];

    supportedTokens.forEach((token) => {
      this.redisService.subscribe('LOGX_DARKORACLE_SERVICE_' + token, (priceUpdate) => {
        client.emit('priceUpdate', { token, priceUpdate: JSON.parse(priceUpdate) });
      });
    });
  }

  async setupRedisSubscription(client: Socket, token: string) {
    this.redisService.subscribe('LOGX_DARKORACLE_SERVICE_' + token, (priceUpdate) => {
      client.emit('priceUpdate', { token, priceUpdate: JSON.parse(priceUpdate) });
    });
  }

  handleConnection(client: Socket) {
    console.log(`Client connected: ${client.id}`);
  }

  handleDisconnect(client: Socket) {
    console.log(`Client disconnected: ${client.id}`);
  }

  @SubscribeMessage('subscribeToPrices')
  async handlePriceSubscription(@MessageBody() data: { token?: string } = {}, @ConnectedSocket() client: Socket) {
    const { token } = data;
    if (token) {
      console.log(`Client ${client.id} subscribed to price updates for token: ${token}`);
      this.setupRedisSubscription(client, token);
    } else {
      console.log(`Client ${client.id} subscribed to price updates for all tokens`);
      this.setupRedisSubscriptions(client);
    }
  }
}
