/**
 * @param forexPair - The forex pair symbol (e.g., "EURUSD", "AUDUSD")
 * @returns The normalized base currency symbol (e.g., "EUR", "AUD")
 */
export function normalizeForexSymbol(forexPair: string): string {
  if (forexPair.endsWith('USD')) {
    return forexPair.substring(0, forexPair.length - 3);
  }

  if (forexPair.length >= 6) {
    return forexPair.substring(0, 3);
  }
  return forexPair;
}

/**
 * @param forexPairs - Array of forex pair symbols
 * @returns Array of normalized base currency symbols
 */
export function normalizeForexPairs(forexPairs: string[]): string[] {
  return forexPairs.map((pair) => normalizeForexSymbol(pair.toUpperCase()));
}

/**
 * @param envVar - Environment variable string (comma-separated pairs)
 * @returns Array of normalized base currency symbols
 */
export function parseAndNormalizeForexPairs(envVar: string | undefined): string[] {
  if (!envVar) {
    return [];
  }

  const forexPairs = envVar
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item !== '');

  return normalizeForexPairs(forexPairs);
}
