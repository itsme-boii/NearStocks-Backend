import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import * as dotenv from 'dotenv';

async function bootstrap() {
  dotenv.config();
  console.log('Starting Dark Oracle on node version ', process.version);
  const app = await NestFactory.create(AppModule.register());
  app.enableCors();
  await app.listen(process.env.PORT || 3000);
}
bootstrap();
