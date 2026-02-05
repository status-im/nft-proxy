# NFT Proxy with Authentication

NFT API proxy with caching and authentication (Basic Auth + Puzzle JWT).

## Architecture

```
[Client] → [nginx-proxy] → [nft-proxy (Go)] → [Alchemy API]
                ↓
         [auth-service (Go)]
```

- **nginx-proxy**: OpenResty with Lua for hybrid authentication
- **nft-proxy**: Go service with 2-level caching (BigCache + KeyDB)
- **auth-service**: Puzzle auth + JWT token generation/validation

## Authentication

### 1. Basic Auth (Simple)

```bash
curl -u admin:admin123 \
  "http://localhost:8080/ethereum/mainnet/nft/v3/getOwnersForContract?contractAddress=0x..."
```

### 2. Puzzle Auth (JWT)

**Step 1: Get puzzle**
```bash
curl http://localhost:8080/auth/puzzle
```

Response:
```json
{
  "challenge": "abc123...",
  "salt": "def456...",
  "difficulty": 1,
  "expires_at": "2026-02-06T12:00:00Z",
  "hmac": "signature",
  "algorithm": "argon2id"
}
```

**Step 2: Solve puzzle (find nonce where Argon2 hash starts with `difficulty` zeros)**

See example solver: `proxy-common/auth/cmd/test-puzzle-auth/main.go`

**Step 3: Submit solution**
```bash
curl -X POST http://localhost:8080/auth/solve \
  -H "Content-Type: application/json" \
  -d '{
    "challenge": "...",
    "salt": "...",
    "nonce": 12345,
    "argon_hash": "00abc...",
    "hmac": "...",
    "expires_at": "..."
  }'
```

Response:
```json
{
  "token": "eyJhbGc...",
  "expires_at": "2026-02-06T12:10:00Z",
  "request_limit": 100
}
```

**Step 4: Use JWT token**
```bash
# In header
curl -H "Authorization: Bearer eyJhbGc..." \
  "http://localhost:8080/ethereum/mainnet/nft/v3/getOwnersForContract?contractAddress=0x..."

# Or in query param
curl "http://localhost:8080/ethereum/mainnet/nft/v3/getOwnersForContract?contractAddress=0x...&token=eyJhbGc..."
```

## Setup

### Quick Start (Local Development)

**One-command setup with embedded credentials:**

```bash
./start-local.sh
```

This automatically:
- ✅ Creates Basic Auth user `test:test`
- ✅ Uses embedded Alchemy API key
- ✅ Starts all services with KeyDB
- ✅ Shows test commands

**Test immediately:**
```bash
# Basic Auth
curl -u test:test "http://localhost:8080/ethereum/mainnet/nft/v3/getOwnersForContract?contractAddress=0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D"

# Get JWT via puzzle auth
curl http://localhost:8080/auth/puzzle | jq
```

### Production Setup

### 1. Prerequisites

- Docker & Docker Compose
- Alchemy API key
- KeyDB/Redis (optional, for L2 cache)

### 2. Configuration

**Environment variables** (create `.env` file):
```bash
ALCHEMY_API_KEY=your_alchemy_api_key_here
CACHE_KEYDB_URL=redis://keydb:6379  # or leave empty to disable L2 cache
```

**Secrets** (already created in `secrets/`):
- `auth_config.json` - Auth service configuration
- `.htpasswd` - Basic auth users (default: `admin:admin123`)

**IMPORTANT**: Change these in production:
```bash
# Generate new JWT secret (64 random chars)
openssl rand -hex 32

# Add/update htpasswd users
htpasswd -c secrets/.htpasswd username

# Update puzzle difficulty (higher = more CPU required)
# Edit secrets/auth_config.json: "puzzle_difficulty": 2
```

### 3. Build & Run

**Local Development:**
```bash
./start-local.sh
```

**Production:**
```bash
# Build and start all services
docker-compose up --build -d

# Check logs
docker-compose logs -f

# Check health
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

### 4. Test Authentication

**Test Basic Auth:**
```bash
curl -u admin:admin123 http://localhost:8080/health
```

**Test Puzzle Auth:**
```bash
# Get puzzle
curl http://localhost:8080/auth/puzzle

# Use test client (from eth-rpc-proxy):
cd proxy-common/auth
go run cmd/test-puzzle-auth/main.go \
  -url http://localhost:8080 \
  -test-endpoint /health
```

## API Endpoints

### NFT Endpoints (require auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/{chain}/{network}/nft/v3/getOwnersForContract` | Get owners of NFT collection |
| GET | `/{chain}/{network}/nft/v3/getNFTsForOwner` | Get NFTs owned by address |
| POST | `/{chain}/{network}/nft/v3/getNFTMetadataBatch` | Batch fetch NFT metadata (max 100) |
| POST | `/{chain}/{network}/nft/v3/getContractMetadataBatch` | Batch fetch contract metadata (max 100) |

Supported chains: `ethereum`, `polygon`, `arbitrum`, `optimism`, `base`  
Networks: `mainnet`, `testnet`

### Auth Endpoints (no auth required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/auth/puzzle` | Get puzzle challenge |
| POST | `/auth/solve` | Submit solution, get JWT |
| GET | `/auth/verify` | Verify JWT token (internal) |
| GET | `/auth/status` | Auth service health |

### Monitoring

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check (no auth) |
| GET | `/metrics` | Prometheus metrics (no auth) |

## Response Headers

- `X-Cache-Status`: `HIT` or `MISS` (cache status)
- `X-Cache-Level`: `1` (BigCache) or `2` (KeyDB)
- `X-RateLimit-Limit`: Max requests per token
- `X-RateLimit-Remaining`: Remaining requests
- `X-Auth-Cache-Status`: `HIT` or `MISS` (JWT cache)

## Configuration Files

### Auth Config (`secrets/auth_config.json`)

```json
{
  "jwt_secret": "change-me",
  "puzzle_difficulty": 1,
  "requests_per_token": 100,
  "token_expiry_minutes": 10,
  "argon2_params": {
    "memory_kb": 16384,
    "time": 4,
    "threads": 4,
    "key_len": 32
  }
}
```

### Cache Config (`cache_config.yaml`)

See existing file for L1/L2 cache settings.

### Cache Rules (`cache_rules.yaml`)

See existing file for endpoint-specific TTL rules.

## Production Checklist

- [ ] Change `jwt_secret` in `secrets/auth_config.json`
- [ ] Update `.htpasswd` with strong passwords
- [ ] Increase `puzzle_difficulty` (2-3 for production)
- [ ] Configure TLS in external nginx (ansible)
- [ ] Set up external KeyDB/Redis for L2 cache
- [ ] Configure rate limiting in external nginx
- [ ] Set up monitoring (Prometheus + Grafana)
- [ ] Enable gzip compression in external nginx

## Troubleshooting

**Auth fails with 401:**
```bash
# Check auth service logs
docker-compose logs auth-service

# Verify JWT manually
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/auth/verify
```

**Cache misses:**
```bash
# Check KeyDB connection
docker-compose logs nft-proxy | grep keydb

# Check cache config
cat cache_config.yaml
```

**nginx errors:**
```bash
# Check nginx logs
docker-compose logs nginx-proxy

# Test nginx config
docker-compose exec nginx-proxy openresty -t
```

## Development

**Rebuild specific service:**
```bash
docker-compose up --build nginx-proxy
```

**Update Lua scripts:**
```bash
# Edit nginx-proxy/lua/**/*.lua
docker-compose restart nginx-proxy
```

**Update auth config:**
```bash
# Edit secrets/auth_config.json
docker-compose restart auth-service nginx-proxy
```

## License

See LICENSE file.
