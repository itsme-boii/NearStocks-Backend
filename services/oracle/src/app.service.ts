import { Injectable } from '@nestjs/common';
import { RedisService } from './redis/redis.service';

@Injectable()
export class AppService {
  constructor(private readonly redisService: RedisService) {}

  async getDarkOraclePrice(token: string): Promise<string> {
    const price = JSON.parse(await this.redisService.get(token));
    return price;
  }

  async getMultipleDarkOraclePrices(tokens: string[]): Promise<{ [token: string]: string }> {
    const prices = await this.redisService.mget(tokens);
    const result = tokens.reduce((acc, token, index) => {
      acc[token] = JSON.parse(prices[index]);
      return acc;
    }, {});
    return result;
  }
}
