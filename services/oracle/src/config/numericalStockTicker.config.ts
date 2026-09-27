export const NUMERICAL_STOCK_TICKER_MAP: Record<string, string> = {
  '700.HK': 'TENCENT',
  '9988.HK': 'ALIBABA',
  '1211.HK': 'BYD',
  '1810.HK': 'XIAOMI',
  '9888.HK': 'BAIDU',
  '3690.HK': 'MEITUAN',
  '2331.HK': 'LINING',
  '1398.HK': 'ICBC',
  '1024.HK': 'KUAISHOU',
};

// Hyperliquid token mapping (API name -> internal name)
// Format: '<dex>:<symbol>' -> 'INTERNAL_NAME'
export const HYPERLIQUID_TOKEN_MAP: Record<string, string> = {
  'xyz:XYZ100': 'XYZ100', // Nasdaq 100 tracking index from trade.xyz
  'xyz:AAPL': 'AAPL', // Hyperliquid perpetual APPLE token  
  'xyz:TSLA': 'TSLA', // Hyperliquid perpetual TESLA token
  'xyz:NVDA': 'NVDA', // Hyperliquid perpetual NVIDIA token
  'xyz:GOOGL': 'GOOGL', // Hyperliquid perpetual GOOGLE token
};
