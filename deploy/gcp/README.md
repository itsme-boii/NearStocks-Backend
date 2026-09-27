# Where env vars and secrets actually go (GCP Cloud Run)

Two different places, never mixed:

## Non-secret config → `deploy/gcp/<service>.env.yaml` (tracked, committed)

Plain values like `NEAR_RPC_URL`, `NEAR_STOCKS_ACCOUNT`, `NEAR_TREASURY_ACCOUNT` (a public account
id, not a key), region names, feature flags. `api-server.env.yaml` here is a worked example — copy
its shape for `engine`, `cron-server`, `balance-server`, `liquidation`, `amm`, `indexer-server`, and
`oracle`, filling in each service's own needs (see each service's `.env.sample` and
`local/common.env` for what it currently reads). `cloudbuild.yaml` passes this file straight to
`gcloud run deploy --env-vars-file`.

## Secrets → GCP Secret Manager (never a file in this repo, not even gitignored)

This is where the sequencer private key goes. Not in code, not in a `.env` file, not pasted
anywhere — GCP Secret Manager is built exactly for this and integrates with Cloud Run natively:
the secret's value never touches disk in the container image or this repo, it's injected at
container start as an env var.

**Run these yourself** (I'm not going to hold or see the key):

```bash
# one-time: create the secret
gcloud secrets create near-sequencer-key --replication-policy=automatic

# add the value — paste the actual "ed25519:..." string when prompted by this command,
# never into a chat, a file in this repo, or anywhere else
gcloud secrets versions add near-sequencer-key --data-file=-
# (paste the key, then Ctrl-D)

# let cron-server's Cloud Run service account read it
gcloud secrets add-iam-policy-binding near-sequencer-key \
  --member="serviceAccount:<cron-server's runtime service account>" \
  --role="roles/secretmanager.secretAccessor"
```

Then wire it into the deploy via `cloudbuild.yaml`'s `_SECRETS` substitution for the `cron-server`
trigger only (it's the only service that needs it):

```
_SECRETS=NEAR_SEQUENCER_PRIVATE_KEY=near-sequencer-key:latest
```

Cloud Run then sets `NEAR_SEQUENCER_PRIVATE_KEY` as a real env var inside the container at
startup, read straight from Secret Manager — `nearchain/kms.go` picks it up via
`os.Getenv("NEAR_SEQUENCER_PRIVATE_KEY")` exactly like it does today locally.

**Worth remembering** (from the earlier audit): `nearchain/kms.go` explicitly documents this env
var as "testnet and local only" — it's built to prefer `NEAR_SEQUENCER_KMS_KEY_ID` (AWS KMS) for
real mainnet, since the sequencer signs every batch continuously and is the highest-exposure key
in the system. That code path is AWS-only today; using GCP means either bridging to AWS KMS with
IAM credentials also stored in Secret Manager, or accepting the plaintext-in-Secret-Manager
approach above as an interim step. Decide which before real trading volume, not after.

Same treatment for every other secret: `DSN` (contains the DB password — treat as a secret, not an
env-vars-file entry), `NEAR_TREASURY_PRIVATE_KEY` (only if api-server signs treasury actions
directly), and any third-party API keys (Tiingo/Finnhub/Allticks for `services/oracle`, if you wire
those in later).

## Quick reference: what's a secret vs. what isn't

| Goes in `*.env.yaml` (committed) | Goes in Secret Manager (never committed) |
|---|---|
| `NEAR_RPC_URL`, `NEAR_STOCKS_ACCOUNT` | `NEAR_SEQUENCER_PRIVATE_KEY` / `NEAR_SEQUENCER_KMS_KEY_ID` |
| `NEAR_TREASURY_ACCOUNT`, `NEAR_SEQUENCER_ACCOUNT` (public account ids) | `NEAR_TREASURY_PRIVATE_KEY` |
| `NEAR_USDC_CONTRACT`, `NEAR_BROKER_ID` | `DSN` (has the DB password embedded) |
| Service URLs (`ENGINE_URL` etc.) | Any third-party API key (Finnhub, Tiingo, Allticks, ...) |
| `CORS_ALLOWED_ORIGINS`, region/repo names | AWS credentials if bridging to AWS KMS |
