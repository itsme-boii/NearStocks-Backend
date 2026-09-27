# Price Aggregator Service

A robust market data service that aggregates real-time token prices from multiple high-liquidity sources and persists the unified price to Redis

## 🚀 Overview

This service ensures price integrity by fetching data from a diverse set of 14 exchanges and data providers. By aggregating these sources, the service minimizes the impact of "flash crashes," outlier pricing, or API downtime from any single provider.

### The Process
1. **Fetch**: Polls prices from CEXs, DEXs, and Financial APIs.
2. **Aggregate**: Computes a unified price point.
3. **Store**: Updates the global state in **Redis** for instant access by downstream services.

### Architecture
1. Contains both reader and writer within same codebase.
2. Reader can be turned on using `IS_GATEWAY=0` and Writer with `IS_GATEWAY=1` within env.
3. Fetch is periodic and more sources can be integrated as adapters.

## 📊 Supported Data Sources

The service currently monitors the following markets:

| Provider | Weight/Sources | Category |
| :--- | :---: | :--- |
| **Tiingo** | 5 | Multi-Asset API |
| **TiingoForex** | 3 | Forex Data |
| **Binance** | 3 | Centralized Exchange |
| **Coinbase** | 2 | Centralized Exchange |
| **OKX** | 2 | Centralized Exchange |
| **Bybit** | 2 | Centralized Exchange |
| **Kraken** | 2 | Centralized Exchange |
| **HTX** | 1 | Centralized Exchange |
| **RedStone** | 1 | Oracle / Web3 |
| **MEXC** | 1 | Centralized Exchange |
| **Gate.io** | 1 | Centralized Exchange |
| **Finnhub** | 1 | TradFi/Crypto API |
| **Allticks** | 1 | Market Data API |
| **Hyperliquid** | 1 | Decentralized Exchange |

## 🛠 Tech Stack

- **Frameworkd**: Nestjs
- **Primary Database**: Redis (In-memory storage)
- **Data Protocols**: REST & WebSockets

### How to run
1. `npm install`
2. `npm run start:dev`

NOTE: Codebase contains .env file but it's just for reference. You might have to copy some vars from production to test properly.
