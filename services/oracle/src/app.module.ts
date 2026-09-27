import { Module, DynamicModule } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { WinstonLogger } from './utils/winston.service';
import { ScheduleModule } from '@nestjs/schedule';
import { PriceService } from './price.service';
import { GatewayService } from './gateway/gateway.service';
import { RedisService } from './redis/redis.service';
import { CantonAuthService } from './canton/canton-auth.service';
import { CantonPricePillService } from './canton/canton-pricepill.service';

@Module({})
export class AppModule {
  static register(): DynamicModule {
    const providers: any[] = [
      WinstonLogger,
      {
        provide: 'LoggerName',
        useValue: 'AppService',
      },
      AppService,
      RedisService,
    ];

    const isGatewayListener = process.env.IS_GATEWAY;
    if (isGatewayListener === '1') {
      providers.push(GatewayService);
    } else {
      providers.push(CantonAuthService);
      providers.push(CantonPricePillService);
      providers.push(PriceService);
    }

    return {
      module: AppModule,
      imports: [ScheduleModule.forRoot()],
      controllers: [AppController],
      providers: providers,
    };
  }
}
