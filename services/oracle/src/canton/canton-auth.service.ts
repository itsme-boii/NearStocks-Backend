import { Injectable } from '@nestjs/common';
import axios from 'axios';
import { WinstonLogger } from '../utils/winston.service';

/**
 * Auth0 client-credentials token management for Canton Ledger API.
 * Mirrors the Go backend pattern in contract/canton/utils.canton.go.
 * Caches the token in-memory with expiry tracking.
 * Token is also invalidated on 401/403 responses from the ledger.
 */
@Injectable()
export class CantonAuthService {
  private logger: WinstonLogger;
  private cachedToken: string | null = null;
  private tokenExpiresAt: number = 0; // Unix timestamp (ms)

  constructor() {
    this.logger = new WinstonLogger('CantonAuthService');
  }

  async getToken(): Promise<string> {
    // Return cached token if still valid (with 60s buffer)
    if (this.cachedToken && Date.now() < this.tokenExpiresAt) {
      return this.cachedToken;
    }

    const tokenUrl = process.env.AUTH0_TOKEN_URL || 'https://arcane-labs.us.auth0.com/oauth/token';
    const clientId = process.env.AUTH0_CLIENT_ID;
    const clientSecret = process.env.AUTH0_CLIENT_SECRET;
    const audience = process.env.AUTH0_AUDIENCE || 'https://canton.network.global';

    if (!clientId || !clientSecret) {
      throw new Error('AUTH0_CLIENT_ID and AUTH0_CLIENT_SECRET must be set');
    }

    this.logger.log(`Fetching Auth0 token from ${tokenUrl}`);

    const response = await axios.post(
      tokenUrl,
      {
        client_id: clientId,
        client_secret: clientSecret,
        audience: audience,
        grant_type: 'client_credentials',
      },
      {
        headers: { 'Content-Type': 'application/json' },
        timeout: 10000,
      },
    );

    const { access_token, expires_in } = response.data;

    if (!access_token) {
      throw new Error('access_token missing in Auth0 response');
    }

    // Cache with 60s buffer before expiry; default to 1 hour if expires_in is missing/invalid
    const defaultExpiresIn = 3600;
    const parsedExpiresIn = Number(expires_in);
    const expiresInSeconds = Number.isFinite(parsedExpiresIn) ? parsedExpiresIn : defaultExpiresIn;

    if (!Number.isFinite(parsedExpiresIn)) {
      this.logger.warn(`Auth0 response missing or invalid expires_in (${expires_in}); using default TTL of ${defaultExpiresIn}s`);
    }

    const ttlMs = Math.max((expiresInSeconds - 60) * 1000, 60000);
    this.cachedToken = access_token;
    this.tokenExpiresAt = Date.now() + ttlMs;

    this.logger.log(`Cached Auth0 token, TTL: ${Math.floor(ttlMs / 1000)}s`);
    return access_token;
  }

  /**
   * Clears the cached token, forcing a fresh fetch on the next getToken() call.
   * Call this when the ledger returns 401 or 403.
   */
  clearToken(): void {
    this.cachedToken = null;
    this.tokenExpiresAt = 0;
    this.logger.warn('Auth0 token cleared (forced re-fetch on next request)');
  }
}
