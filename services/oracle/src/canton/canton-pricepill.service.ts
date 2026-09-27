import { Injectable } from '@nestjs/common';
import axios, { AxiosInstance } from 'axios';
import { WinstonLogger } from '../utils/winston.service';
import { CantonAuthService } from './canton-auth.service';
import { getCantonPricePillInterfaceId, CANTON_FEED_ID_MAP, CANTON_FEED_ID_REVERSE_MAP } from './canton-pricepill.constants';

/**
 * Reads RedStone PricePill contracts from Canton Ledger API.
 *
 * Uses a two-phase approach for efficiency:
 *   1. Initial full load via /v2/state/active-contracts (one-time at startup)
 *   2. Delta updates via /v2/updates (long-poll loop, ~10-15s cycles)
 *
 * Falls back to full active-contracts refresh on errors or periodically as a safety net.
 */

export interface PricePillData {
  feedId: string;
  price: number;
  dataTimestamp: number;
  contractId: string;
  templateId?: string;
}

/** How often to do a full active-contracts refresh as a safety net (ms) */
const FULL_REFRESH_INTERVAL_MS = 20 * 60 * 1000; // 20 minutes

/** Max idle time before Canton returns empty (ms).
 *  Data arrives every ~10-13s, but total request time (data arrival + idle wait)
 *  must stay under the nginx proxy timeout (~20s). Keep this short. */
const STREAM_IDLE_TIMEOUT_MS = 5_000;

/** Max consecutive stream errors before falling back to full refresh */
const MAX_STREAM_ERRORS = 3;

/** Max updates to fetch per /v2/updates call.
 *  Keep low so the request returns quickly after the first batch arrives. */
const STREAM_LIMIT = 5;

@Injectable()
export class CantonPricePillService {
  private logger: WinstonLogger;
  private readonly pricePillInterfaceId = getCantonPricePillInterfaceId();

  // Pre-computed static config (set once in constructor)
  private readonly ledgerApiUrl: string;
  private readonly partyId: string;
  private readonly cantonFeedIds: string[];
  private readonly requestedFeedIdSet: Set<string>;
  private readonly filtersByParty: Record<string, any>;

  // Streaming state
  private lastOffset: number | null = null;
  private lastFullRefreshAt = 0;
  private consecutiveStreamErrors = 0;
  private cachedPrices: Record<string, number> = {};
  private cachedPillData: Record<string, PricePillData> = {};
  private running = false;
  private stopRequested = false;
  private httpClient: AxiosInstance | null = null;

  // Staleness checking
  private stalenessCache: Record<string, { stale: boolean; checkedAt: number }> = {};
  private readonly stalenessCheckIntervalMs: number;
  private readonly localStaleThresholdMs: number;

  constructor(private readonly authService: CantonAuthService) {
    this.logger = new WinstonLogger('CantonPricePillService');

    this.ledgerApiUrl = process.env.CANTON_LEDGER_API_URL || '';
    this.partyId = process.env.CANTON_PARTY_ID || '';

    const requestedFeedIds = (process.env.CANTON_FEED_IDS || '')
      .split(',')
      .map((feedId) => feedId.trim())
      .filter(Boolean);

    this.cantonFeedIds = requestedFeedIds.map((id) => CANTON_FEED_ID_MAP[id] ?? id);
    this.requestedFeedIdSet = new Set(this.cantonFeedIds);

    this.stalenessCheckIntervalMs = Number(process.env.CANTON_STALENESS_CHECK_INTERVAL_MS || '30000');
    this.localStaleThresholdMs = Number(process.env.CANTON_LOCAL_STALE_THRESHOLD_MS || '300000');

    // Pre-build the static filter object (reused for every request)
    this.filtersByParty = this.partyId
      ? {
          [this.partyId]: {
            cumulative: [
              {
                identifierFilter: {
                  InterfaceFilter: {
                    value: {
                      interfaceId: this.pricePillInterfaceId,
                      includeInterfaceView: true,
                      includeCreatedEventBlob: false,
                    },
                  },
                },
              },
            ],
          },
        }
      : {};
  }

  /**
   * Start the streaming update loop. Called once from PriceService.
   * Runs indefinitely until stopStreaming() is called.
   */
  async startStreaming(): Promise<void> {
    if (this.running) return;
    if (!this.ledgerApiUrl || !this.partyId) {
      this.logger.error('Canton config missing: CANTON_LEDGER_API_URL or CANTON_PARTY_ID', '');
      return;
    }

    this.running = true;
    this.stopRequested = false;
    this.logger.log('Starting Canton PricePill streaming loop');

    while (!this.stopRequested) {
      try {
        await this.ensureHttpClient();

        const needsFullRefresh =
          this.lastOffset === null ||
          Date.now() - this.lastFullRefreshAt > FULL_REFRESH_INTERVAL_MS ||
          this.consecutiveStreamErrors >= MAX_STREAM_ERRORS;

        if (needsFullRefresh) {
          await this.doFullRefresh();
        } else {
          await this.doStreamUpdate();
        }

        // Check staleness inline after each poll cycle
        await this.checkAllStaleness();
      } catch (error: any) {
        this.logger.error('Canton streaming loop error', error?.message || error);
        this.consecutiveStreamErrors++;
        // Brief pause before retry on unexpected errors
        await this.sleep(2000);
      }
    }

    this.running = false;
    this.logger.log('Canton PricePill streaming loop stopped');
  }

  /** Stop the streaming loop gracefully */
  stopStreaming(): void {
    this.stopRequested = true;
  }

  /** Get the latest cached prices (called by PriceService on its own interval) */
  getCachedPrices(): Record<string, number> {
    return { ...this.cachedPrices };
  }

  /** Get the full cached PricePillData including contractId and dataTimestamp */
  getCachedPillData(): Record<string, PricePillData> {
    return { ...this.cachedPillData };
  }

  // ─── Staleness Checking ──────────────────────────────────────────────────

  /**
   * Check whether a feed's data is stale. Uses two layers:
   * 1. Contract-level IsDataStale result (cached, refreshed periodically)
   * 2. Local dataTimestamp age check (belt-and-suspenders)
   * Returns true if stale or no data exists (fail-safe to secondary).
   */
  isFeedStale(canonicalId: string): boolean {
    const pillData = this.cachedPillData[canonicalId];
    if (!pillData) return true;

    // Layer 1: Contract-level staleness (if we have a cached result)
    const cached = this.stalenessCache[canonicalId];
    if (cached?.stale) return true;

    // Layer 2: Local dataTimestamp check
    // Auto-detect: if timestamp < 10^12, it's seconds; otherwise milliseconds
    const tsMs = pillData.dataTimestamp < 1e12 ? pillData.dataTimestamp * 1000 : pillData.dataTimestamp;
    const ageMs = Date.now() - tsMs;
    if (ageMs > this.localStaleThresholdMs) {
      this.logger.warn(`Canton PricePill local staleness detected for ${canonicalId}: ` + `dataTimestamp=${pillData.dataTimestamp}, age=${Math.round(ageMs / 1000)}s, threshold=${this.localStaleThresholdMs / 1000}s`);
      return true;
    }

    return false;
  }

  /**
   * Exercise IsDataStale for cached contracts from the streaming loop.
   * Invoked after each streaming poll cycle, but skips feeds checked recently
   * based on `stalenessCheckIntervalMs`.
   */
  async checkAllStaleness(): Promise<void> {
    if (!this.httpClient) return;

    const feedsToCheck: { canonicalId: string; contractId: string }[] = [];
    for (const [canonicalId, pillData] of Object.entries(this.cachedPillData)) {
      if (!pillData.contractId) continue;

      const cached = this.stalenessCache[canonicalId];
      if (cached && Date.now() - cached.checkedAt < this.stalenessCheckIntervalMs) {
        continue;
      }
      feedsToCheck.push({ canonicalId, contractId: pillData.contractId });
    }

    if (feedsToCheck.length === 0) return;

    const results = await Promise.allSettled(
      feedsToCheck.map(({ canonicalId, contractId }) =>
        this.checkIsDataStale(canonicalId, contractId).then((isStale) => ({ canonicalId, contractId, isStale })),
      ),
    );

    for (const result of results) {
      if (result.status === 'fulfilled') {
        const { canonicalId, contractId, isStale } = result.value;
        this.stalenessCache[canonicalId] = { stale: isStale, checkedAt: Date.now() };
        if (isStale) {
          this.logger.warn(`Canton PricePill IsDataStale=true for ${canonicalId} (contract=${contractId})`);
        }
      }
      // rejected results are already logged inside checkIsDataStale; fail-safe defaults apply
    }
  }

  private async checkIsDataStale(canonicalId: string, contractId: string): Promise<boolean> {
    try {
      const token = await this.authService.getToken();
      this.updateHttpClientToken(token);

      const response = await this.httpClient!.post(
        '/v2/commands/submit-and-wait-for-transaction-tree',
        {
          commandId: `staleness-check-${canonicalId}-${Date.now()}`,
          applicationId: 'canton-pricepill-staleness',
          actAs: [this.partyId],
          commands: [
            {
              ExerciseCommand: {
                templateId: this.pricePillInterfaceId,
                contractId,
                choice: 'IsDataStale',
                choiceArgument: { caller: this.partyId },
              },
            },
          ],
        },
        { timeout: 10_000 },
      );

      // Parse the exercise result from the transaction tree.
      // The response shape matches the proof script's submit-and-wait-for-transaction-tree:
      //   response.data.transactionTree.eventsById -> { ExercisedTreeEvent.value.exerciseResult }
      const tree = response.data?.transactionTree;
      const eventsById = tree?.eventsById || {};
      const exercised = Object.values(eventsById)
        .map((event: any) => event?.ExercisedTreeEvent?.value)
        .find((value: any) => value?.choice === 'IsDataStale');

      const exerciseResult = exercised?.exerciseResult;

      // IsDataStale returns a boolean (wire alias: IsDataStaleBool)
      const isStale = exerciseResult === true || exerciseResult === 'true';

      this.logger.log(`IsDataStale check for ${canonicalId}: result=${exerciseResult}, stale=${isStale}`);
      return isStale;
    } catch (error: any) {
      const status = error?.response?.status;
      const body = error?.response?.data;

      if (status === 404) {
        // Contract was archived and replaced by a newer one — data is actively flowing, not stale
        this.logger.log(`IsDataStale: contract archived (404) for ${canonicalId}, treating as fresh`);
        return false;
      }

      this.logger.error(`IsDataStale check failed for ${canonicalId} (contract=${contractId})`, `status=${status}, error=${body?.cause || error?.message || error}`);
      // Fail-safe: treat as stale so we fall back to secondary
      return true;
    }
  }

  /**
   * Legacy method: single-shot fetch via active-contracts.
   * Kept for backwards compatibility but startStreaming() + getCachedPrices() is preferred.
   */
  async fetchPricePillPrices(): Promise<Record<string, number>> {
    if (!this.ledgerApiUrl || !this.partyId) {
      this.logger.error('Canton config missing: CANTON_LEDGER_API_URL or CANTON_PARTY_ID', '');
      return {};
    }

    await this.ensureHttpClient();
    await this.doFullRefresh();
    return { ...this.cachedPrices };
  }

  // ─── Full Refresh (active-contracts) ───────────────────────────────────────

  private async doFullRefresh(): Promise<void> {
    const token = await this.authService.getToken();
    this.updateHttpClientToken(token);

    // Step 1: Get ledger offset
    const offset = await this.getLedgerOffset();
    if (offset === null) return;

    // Step 2: Get all active PricePill contracts
    const contracts = await this.getActiveContracts(offset);
    const pills = this.parseContracts(contracts);

    // Step 3: Build price map from newest per feed
    const prices = this.buildPriceMap(pills);

    if (Object.keys(prices).length > 0) {
      this.cachedPrices = prices;
      this.lastOffset = typeof offset === 'string' ? parseInt(offset, 10) : offset;
      this.lastFullRefreshAt = Date.now();
      this.consecutiveStreamErrors = 0;
      this.logger.log(`Full refresh: ${Object.keys(prices).length} prices, offset=${this.lastOffset}`);
    } else {
      this.logger.warn('Full refresh returned empty prices — keeping previous cache');
    }
  }

  // ─── Stream Update (/v2/updates) ───────────────────────────────────────────

  private async doStreamUpdate(): Promise<void> {
    if (this.lastOffset === null) {
      await this.doFullRefresh();
      return;
    }

    try {
      const response = await this.httpClient!.post(
        `/v2/updates?limit=${STREAM_LIMIT}&stream_idle_timeout_ms=${STREAM_IDLE_TIMEOUT_MS}`,
        {
          beginExclusive: this.lastOffset,
          updateFormat: {
            includeTransactions: {
              transactionShape: 'TRANSACTION_SHAPE_ACS_DELTA',
              eventFormat: {
                filtersByParty: this.filtersByParty,
                verbose: true,
              },
            },
          },
        },
        { timeout: STREAM_IDLE_TIMEOUT_MS + 10_000 },
      );

      const updates = Array.isArray(response.data) ? response.data : [];

      // Count event types for logging
      let createdCount = 0;
      let archivedCount = 0;
      let txCount = 0;

      if (updates.length === 0) {
        this.logger.log(`Stream poll: no new updates (offset=${this.lastOffset})`);
        this.consecutiveStreamErrors = 0;
        return;
      }

      let maxOffset = this.lastOffset;
      let pricesUpdated = false;
      const changedFeeds: string[] = [];

      for (const update of updates) {
        const txValue = update?.update?.Transaction?.value;
        if (!txValue) {
          // Could be an OffsetCheckpoint — extract offset if present
          const checkpoint = update?.update?.OffsetCheckpoint?.value;
          if (checkpoint?.offset != null) {
            const cpOffset = Number(checkpoint.offset);
            if (cpOffset > maxOffset) maxOffset = cpOffset;
          }
          continue;
        }

        txCount++;
        const txOffset = Number(txValue.offset);
        if (txOffset > maxOffset) maxOffset = txOffset;

        const events = txValue.events || [];
        for (const event of events) {
          if (event?.ArchivedEvent) {
            archivedCount++;
            continue;
          }

          const created = event?.CreatedEvent;
          if (!created) continue;
          createdCount++;

          const pill = this.parseOneContract(created);
          if (!pill) continue;

          // Only update if this feed is in our requested set (or we want all)
          if (this.requestedFeedIdSet.size > 0 && !this.requestedFeedIdSet.has(pill.feedId)) continue;

          const canonicalId = CANTON_FEED_ID_REVERSE_MAP[pill.feedId] ?? pill.feedId;
          const existingPrice = this.cachedPrices[canonicalId];

          // Only update if this is actually newer data
          if (existingPrice === undefined || pill.price !== existingPrice) {
            this.cachedPrices[canonicalId] = pill.price;
            this.cachedPillData[canonicalId] = pill;
            pricesUpdated = true;
            changedFeeds.push(`${canonicalId}=${pill.price}`);
          }
        }
      }

      this.lastOffset = maxOffset;
      this.consecutiveStreamErrors = 0;

      this.logger.log(
        `Stream received: ${updates.length} updates, ${txCount} txns, ${createdCount} created, ${archivedCount} archived, offset=${this.lastOffset}` +
          (pricesUpdated ? ` | changed: ${changedFeeds.join(', ')}` : ' | no price changes'),
      );
    } catch (error: any) {
      const status = error?.response?.status;

      if (status === 401 || status === 403) {
        this.logger.warn(`Stream returned ${status}, refreshing token`);
        this.authService.clearToken();
        const freshToken = await this.authService.getToken();
        this.updateHttpClientToken(freshToken);
        this.consecutiveStreamErrors++;
        return;
      }

      if (status === 503) {
        // Proxy timeout — normal when no data arrives within the idle window
        this.logger.warn('Stream 503 (proxy timeout) — retrying');
        return;
      }

      this.consecutiveStreamErrors++;
      const details = error?.response?.data || error?.message || error;
      this.logger.error(`Stream error (${this.consecutiveStreamErrors}/${MAX_STREAM_ERRORS})`, details);

      if (this.consecutiveStreamErrors >= MAX_STREAM_ERRORS) {
        this.logger.warn('Too many stream errors — will do full refresh on next cycle');
      }

      await this.sleep(1000);
    }
  }

  // ─── HTTP Client Management ────────────────────────────────────────────────

  /** Pre-flight: ensure we can obtain an auth token. Resolves once token is available. */
  async waitForAuth(): Promise<void> {
    await this.authService.getToken();
  }

  private async ensureHttpClient(): Promise<void> {
    if (this.httpClient) return;

    const token = await this.authService.getToken();
    this.httpClient = axios.create({
      baseURL: this.ledgerApiUrl,
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      timeout: 20_000,
      // Keep-alive for connection reuse across both HTTP and HTTPS endpoints
      httpAgent: new (require('http').Agent)({ keepAlive: true }),
      httpsAgent: new (require('https').Agent)({ keepAlive: true }),
    });
  }

  private updateHttpClientToken(token: string): void {
    if (this.httpClient) {
      this.httpClient.defaults.headers['Authorization'] = `Bearer ${token}`;
    }
  }

  // ─── Network Calls ─────────────────────────────────────────────────────────

  private async getLedgerOffset(): Promise<number | string | null> {
    try {
      const response = await this.httpClient!.get('/v2/state/ledger-end');
      const offset = response.data?.offset;
      if (offset == null) {
        this.logger.warn('Ledger-end returned null/undefined offset');
        return null;
      }
      return offset;
    } catch (error: any) {
      const status = error?.response?.status;

      if (status === 401 || status === 403) {
        this.logger.warn(`Ledger-end returned ${status}, clearing token and retrying once`);
        this.authService.clearToken();
        try {
          const freshToken = await this.authService.getToken();
          this.updateHttpClientToken(freshToken);
          const retryResponse = await this.httpClient!.get('/v2/state/ledger-end');
          return retryResponse.data?.offset ?? null;
        } catch (retryError: any) {
          this.logger.error('Failed to get ledger offset after token refresh', retryError?.message || retryError);
          return null;
        }
      }

      this.logger.error('Failed to get ledger offset', error?.message || error);
      return null;
    }
  }

  private async getActiveContracts(offset: number | string): Promise<any[]> {
    try {
      const response = await this.httpClient!.post(
        '/v2/state/active-contracts',
        {
          filter: { filtersByParty: this.filtersByParty },
          verbose: true,
          activeAtOffset: offset,
        },
        { timeout: 15_000 },
      );
      return Array.isArray(response.data) ? response.data : [];
    } catch (error: any) {
      const status = error?.response?.status;

      if (status === 401 || status === 403) {
        this.logger.warn(`Active-contracts returned ${status}, clearing token and retrying once`);
        this.authService.clearToken();
        try {
          const freshToken = await this.authService.getToken();
          this.updateHttpClientToken(freshToken);
          const retryResponse = await this.httpClient!.post(
            '/v2/state/active-contracts',
            {
              filter: { filtersByParty: this.filtersByParty },
              verbose: true,
              activeAtOffset: offset,
            },
            { timeout: 15_000 },
          );
          return Array.isArray(retryResponse.data) ? retryResponse.data : [];
        } catch (retryError: any) {
          this.logger.error('Failed to get active contracts after token refresh', retryError?.message || retryError);
          return [];
        }
      }

      const details = error?.response?.data ? `status=${status} body=${JSON.stringify(error.response.data)}` : error?.message || error;
      this.logger.error('Failed to get active PricePill contracts', details);
      return [];
    }
  }

  // ─── Contract Parsing ──────────────────────────────────────────────────────

  private parseContracts(contracts: any[]): PricePillData[] {
    const pills: PricePillData[] = [];
    for (const contract of contracts) {
      const activeContract =
        contract?.contractEntry?.JsActiveContract ||
        contract?.contractEntry?.jsActiveContract ||
        contract?.JsActiveContract ||
        contract;

      const pill = this.parseOneContract(activeContract?.createdEvent || activeContract);
      if (pill) pills.push(pill);
    }
    return pills;
  }

  private parseOneContract(source: any): PricePillData | null {
    try {
      const args = source?.createArgument;
      if (!args) return null;

      const feedIdArray = args.feedId || args.feed_id || [];
      const feedId = this.asciiArrayToString(feedIdArray);
      if (!feedId) return null;

      const rawValue = args.priceData?.value || args.value || args.price || '0';
      const price = typeof rawValue === 'string' ? parseFloat(rawValue) : Number(rawValue);
      if (isNaN(price) || price <= 0) return null;

      const dataTimestamp = parseInt(args.priceData?.timestamp || args.dataTimestamp || args.data_timestamp || '0', 10);
      const contractId = source?.contractId || '';
      const templateId = source?.templateId || undefined;

      return { feedId, price, dataTimestamp, contractId, templateId };
    } catch (error: any) {
      this.logger.error('Failed to parse PricePill contract', error?.message || error);
      return null;
    }
  }

  private buildPriceMap(pills: PricePillData[]): Record<string, number> {
    // Filter by requested feeds
    const filtered =
      this.requestedFeedIdSet.size > 0 ? pills.filter((p) => this.requestedFeedIdSet.has(p.feedId)) : pills;

    // Keep newest per feed
    const newestByFeed: Record<string, PricePillData> = {};
    for (const pill of filtered) {
      const existing = newestByFeed[pill.feedId];
      if (!existing || pill.dataTimestamp > existing.dataTimestamp) {
        newestByFeed[pill.feedId] = pill;
      }
    }

    // Remap to canonical names and persist full pill data
    const prices: Record<string, number> = {};
    const pillData: Record<string, PricePillData> = {};
    for (const [feedId, pill] of Object.entries(newestByFeed)) {
      const canonicalId = CANTON_FEED_ID_REVERSE_MAP[feedId] ?? feedId;
      prices[canonicalId] = pill.price;
      pillData[canonicalId] = pill;
    }
    this.cachedPillData = pillData;
    return prices;
  }

  private asciiArrayToString(arr: any): string {
    if (!Array.isArray(arr)) return '';
    try {
      return arr.map((code: number | string) => String.fromCharCode(Number(code))).join('');
    } catch {
      return '';
    }
  }

  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
