export const DEFAULT_CANTON_PRICEPILL_INTERFACE_ID = '#redstone-price-pill-v12:IRedStonePricePill:IRedStonePricePill';

export function getCantonPricePillInterfaceId(): string {
  return process.env.CANTON_PRICEPILL_INTERFACE_ID?.trim() || DEFAULT_CANTON_PRICEPILL_INTERFACE_ID;
}

/**
 * Maps canonical oracle token names to the actual feed IDs used in Canton PricePill contracts.
 * Stocks are published with the ---24_7 suffix for 24/7 synthetic markets.
 * Tokens not listed here use their name as-is.
 */
export const CANTON_FEED_ID_MAP: Record<string, string> = {
  AAPL: 'AAPL---24_7',
  TSLA: 'TSLA---24_7',
  NVDA: 'NVDA---24_7',
  GOOGL: 'GOOGL---24_7',
  XYZ100: 'XYZ100---PERP',
};

/** Reverse map: Canton feed ID → canonical oracle token name */
export const CANTON_FEED_ID_REVERSE_MAP: Record<string, string> = Object.fromEntries(
  Object.entries(CANTON_FEED_ID_MAP).map(([canonical, cantonId]) => [cantonId, canonical]),
);
