## Crypto-Dashboard

A full‑stack, real‑time crypto prices dashboard.

- **Frontend**: React + TypeScript + Vite
- **Backend**: Go (REST API)
- **Database**: Apache Cassandra (time‑series table per coin)
- **Ingestion**: Scheduled fetch from CoinGecko
- **AI Query**: Natural language to CQL via OpenAI

### Features
- **Live prices** and historical charts for popular coins
- **Analytics**: average, min/max range, volatility (stddev), and regression‑based trend
- **Top movers** over a selectable window
- **AI query** endpoint that turns natural‑language questions into CQL and returns rows

---

## Architecture

- `Backend/crypto/backend/main.go`: HTTP API server on port `8000`, CORS to `http://localhost:5173`.
- `Backend/crypto/backend/crypto.go`: Ingestion worker. Every minute pulls prices from CoinGecko and inserts into Cassandra.
- `Backend/crypto/backend/analytics.go`: Volatility, trend, and top movers endpoints.
- `Backend/crypto/backend/AI.go`: `/ask` endpoint calling OpenAI to generate CQL.
- `Frontend/crypto`: React app (Vite). Talks to API at `http://localhost:8000` (see `src/services/api.ts`).
- `Database/Create_Crypto_table.cql`: Table DDL.

Data model (Cassandra):

```sql
CREATE TABLE iot_data.crypto_price_by_coin (
  coin_id text,
  timestamp timestamp,
  price_usd double,
  PRIMARY KEY (coin_id, timestamp)
) WITH CLUSTERING ORDER BY (timestamp DESC);
```

Notes:
- Several queries use `ALLOW FILTERING` for simplicity. For production, model additional tables or materialized views to avoid filtering.
- The ingestion worker connects to Cassandra at `host.docker.internal` (for Docker). The HTTP server connects to `127.0.0.1`. Adjust as needed.

---

## Prerequisites
- Go 1.22+
- Node.js 18+ and npm
- Apache Cassandra 4.x (local or remote)
- Optional: Docker Desktop (to run the ingestion worker in a container)
- For AI endpoint: an OpenAI API key

---

## Setup: Database
1) Start Cassandra locally (or point to a remote cluster).

2) Create keyspace (if not already present):

```sql
CREATE KEYSPACE IF NOT EXISTS iot_data
WITH REPLICATION = {
  'class': 'SimpleStrategy',
  'replication_factor': 1
};
```

3) Create the table:

```bash
cqlsh -f Database/Create_Crypto_table.cql
```

---

## Running: Backend

The backend consists of two programs: the HTTP API server and the ingestion worker.

### 1) HTTP API server (port 8000)

PowerShell (Windows):

```powershell
cd Backend/crypto/backend
go run .\main.go .\analytics.go .\AI.go
```

This server expects Cassandra at `127.0.0.1` and the keyspace `iot_data`.

If you encounter a "redeclared: session" error, ensure there is only a single `var session *gocql.Session` declaration (keep it in `main.go`).

### 2) Ingestion worker (fetch prices every minute)

PowerShell (Windows):

```powershell
cd Backend/crypto/backend
go run .\crypto.go
```

This worker, as written, connects to Cassandra at `host.docker.internal`. If you are not running it in Docker, change that host to `127.0.0.1` (or your Cassandra IP) in `crypto.go`.

### Optional: Run the worker in Docker

```powershell
cd Backend/crypto/backend
docker build -t crypto-ingestor .
docker run --rm --name crypto-ingestor ^
  --network host ^
  crypto-ingestor
```

Notes:
- The provided `Dockerfile` builds and runs the ingestion worker (`crypto.go`). It does not expose an HTTP port.
- Ensure the container can reach Cassandra. On Windows with Docker Desktop, `host.docker.internal` resolves to the host.

---

## Running: Frontend

PowerShell (Windows):

```powershell
cd Frontend/crypto
npm install
npm run dev
```

The app starts on `http://localhost:5173` and calls the backend at `http://localhost:8000` (see `src/services/api.ts`).

Build for production:

```powershell
npm run build
npm run preview
```

---

## Environment variables

Required only if you use the AI endpoint (`/ask`):

```powershell
$env:OPENAI_API_KEY = "sk-..."
```

Then restart the HTTP server so `AI.go` can read it.

---

## API Reference (HTTP)
Base URL: `http://localhost:8000`

- `GET /latest/{coin_id}` → Latest `PriceData`
- `GET /history/{coin_id}?minutes={n}` → Array of `PriceData`
- `GET /average/{coin_id}?start={RFC3339}&end={RFC3339}` → Average over range
- `GET /at/{coin_id}?timestamp={RFC3339}` → Price nearest at/before timestamp
- `GET /range/{coin_id}?start={RFC3339}&end={RFC3339}` → Min/Max over range
- `GET /coins` → Array of available `coin_id`
- `GET /volatility/{coin_id}?start={RFC3339}&end={RFC3339}` → Stddev and mean
- `GET /trend/{coin_id}?start={RFC3339}&end={RFC3339}` → Linear regression slope, label
- `GET /top-movers?minutes={n}` → Largest absolute % movers since T‑n minutes
- `POST /ask` (text/plain) → Array of rows answering NL question via generated CQL

Types (as returned to the frontend):

```ts
// Price
interface PriceData {
  coin_id: string
  timestamp: string // ISO
  price_usd: number
}

// Average
interface AveragePriceData {
  coin_id: string
  average: number
  data_points: number
  start: string
  end: string
}

// Range
interface PriceRangeData {
  coin_id: string
  min: number
  max: number
  start: string
  end: string
}

// Volatility
interface VolatilityData {
  coin_id: string
  start: string
  end: string
  stddev_price: number
  mean_price: number
  data_points: number
}

// Trend
interface TrendData {
  coin_id: string
  slope: number
  trend: string // Uptrend | Downtrend | Sideways
  data_points: number
  start: string
  end: string
}

// Top Movers
interface TopMoverData {
  coin_id: string
  start_price: number
  end_price: number
  percent_change: number
}
```

Example requests:

```bash
curl http://localhost:8000/latest/bitcoin
curl "http://localhost:8000/history/bitcoin?minutes=120"
curl "http://localhost:8000/average/bitcoin?start=2024-01-01T00:00:00Z&end=2024-01-02T00:00:00Z"
curl -X POST -H "Content-Type: text/plain" --data "show last 10 bitcoin prices" http://localhost:8000/ask
```

---

## Frontend development

Key files:
- `src/services/api.ts`: All API calls and base URL
- `src/components/*`: UI components (`Header`, `PriceCard`, `ChartCard`, `StatsCard`, `AdvancedAnalytics`, `CoinManager`)
- `src/App.tsx`: Main dashboard orchestration

Scripts (`Frontend/crypto/package.json`):
- `npm run dev` – start dev server
- `npm run build` – type‑check and build
- `npm run preview` – preview built app
- `npm run lint` – run ESLint

---

## Troubleshooting
- If the frontend loads but data is empty, verify the ingestion worker is running and Cassandra has rows in `iot_data.crypto_price_by_coin`.
- CORS errors: the backend only allows `http://localhost:5173`. Adjust in `main.go` if you use a different origin.
- Time parameters must be RFC3339, e.g. `2024-01-01T00:00:00Z`.
- The `/ask` endpoint requires `$env:OPENAI_API_KEY` and outbound HTTPS connectivity.
- If running the worker in Docker, ensure it can reach Cassandra (`host.docker.internal` on Windows/Mac). On Linux, use the host IP or a user‑defined bridge network.

---

