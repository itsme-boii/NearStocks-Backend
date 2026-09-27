import { Inject, Injectable, Scope } from '@nestjs/common';
import { createLogger, format, transports, Logger } from 'winston';

@Injectable({ scope: Scope.TRANSIENT })
export class WinstonLogger {
  private winstonLogger: Logger;

  constructor(@Inject('LoggerName') private serviceName: string) {
    this.winstonLogger = createLogger({
      format: format.combine(
        format.errors({ stack: true }),
        format.colorize(),
        format.timestamp(),
        format.printf((info) => `${info.timestamp} ${info.level} [${this.serviceName}]: ${info.message}`),
      ),
      transports: [new transports.Console()],
    });
  }

  log(message: string) {
    this.winstonLogger.info(message);
  }

  error(message: string, trace: string) {
    this.winstonLogger.error(message + trace);
  }

  warn(message: string) {
    this.winstonLogger.warn(message);
  }

  debug(message: string) {
    this.winstonLogger.debug(message);
  }

  verbose(message: string) {
    this.winstonLogger.verbose(message);
  }
}
