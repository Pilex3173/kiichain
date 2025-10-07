# Rate Limiter Module

The **Rate Limiter module** enforces transfer rate limits on specific token denominations or channels to enhance security and maintain chain stability.

## Overview
This module restricts cross-chain or inter-module transfers based on predefined thresholds to prevent sudden liquidity drains or bridge exploitation.

## Key Features
- Configurable per-token or per-channel transfer limits.
- Supports both inbound and outbound rate control.
- Integrates with IBC and governance parameters.
- Provides query endpoints for monitoring limit usage.

## Main Functions
- **SetRateLimit** — Defines a rate limit for a token or channel.
- **RemoveRateLimit** — Deletes existing rate configurations.
- **QueryRateLimit** — Fetches limit data and usage status.

## References
- [Rate Limiter Overview (Stride)](https://github.com/Stride-Labs/ibc-rate-limiting)
- Referenced in `go.mod`
