import { Injectable, OnModuleDestroy, OnModuleInit } from '@nestjs/common';
import { createClient, RedisClientType } from 'redis';
import { WinstonLogger } from 'src/utils/winston.service';

const PREFIX = 'LOGX_DARKORACLE_SERVICE_';

@Injectable()
export class RedisService implements OnModuleInit, OnModuleDestroy {
  private client: RedisClientType;
  private subscriber: RedisClientType;
  private logger: WinstonLogger;

  constructor() {
    this.logger = new WinstonLogger('RedisService');
  }

  async onModuleInit() {
    this.logger.log('Creating Redis Client');
    this.client = createClient({
      socket: {
        port: Number(process.env.REDIS_PORT),
        host: process.env.REDIS_HOST,
      },
      password: process.env.REDIS_PASSWORD,
    });

    this.client.on('connect', () => {
      this.logger.log('Connected to Redis');
    });

    this.client.on('error', (err) => {
      this.logger.error('Issue while connecting to Redis:', err);
    });

    try {
      await this.client.connect();
    } catch (err) {
      this.logger.error('Issue while connecting to Redis:', err);
    }
  }

  async set(key: string, value: string) {
    await this.client.set(PREFIX + key, value);
    // Publish the new price to a channel named after the token
    this.client.publish(PREFIX + key, value);
  }

  async get(key: string) {
    return this.client.get(PREFIX + key);
  }

  async mget(keys: string[]): Promise<string[]> {
    const prefixedKeys = keys.map((key) => PREFIX + key);
    return this.client.mGet(prefixedKeys);
  }

  async subscribe(channel: string, callback: (message: string) => void) {
    if (!this.subscriber) {
      this.subscriber = this.client.duplicate(); // Create a single subscriber client
      await this.subscriber.connect();
    }

    try {
      await this.subscriber.subscribe(channel, (message) => {
        callback(message);
      });
    } catch (err) {
      this.logger.error('Issue while subscribing to channel:', err);
    }
  }

  onModuleDestroy() {
    this.logger.log('Closing Redis Client...');
    if (this.subscriber) {
      this.subscriber.quit();
    }
    this.client.quit();
  }
}
