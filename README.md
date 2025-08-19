# Crypto-Dashboard

A full-stack, real-time cryptocurrency prices dashboard.

- **Frontend:** React + TypeScript + Vite
- **Backend:** Go (REST API)
- **Database:** Apache Cassandra (time-series table per coin)
- **Ingestion:** Scheduled fetch from CoinGecko
- **AI Query:** Natural language to CQL via OpenAI

---

## Features

- Live prices and historical charts for popular coins
- Analytics: average, min/max range, volatility (stddev), and regression-based trend
- Top movers over a selectable window
- AI query endpoint: natural language to CQL and results

---

## Architecture

- [`Backend/crypto/backend/main.go`](Backend/crypto/backend/main.go): HTTP API server (port 8000), CORS to `http://localhost:5173`
- [`Backend/crypto/backend/crypto.go`](Backend/crypto/backend/crypto.go): Ingestion worker (fetches prices from CoinGecko and inserts into Cassandra)
- [`Backend/crypto/backend/analytics.go`](Backend/crypto/backend/analytics.go): Volatility, trend, and top movers endpoints
- [`Backend/crypto/backend/AI.go`](Backend/crypto/backend/AI.go): `/ask` endpoint (natural language to CQL via OpenAI)
- [`Frontend/crypto`](Frontend/crypto): React app (see [`src/services/api.ts`](Frontend/crypto/src/services/api.ts))
- [`Database/Create_Crypto_table.cql`](Database/Create_Crypto_table.cql): Table DDL

---

## Prerequisites

- Go 1.22+
- Node.js 18+ and npm
- Apache Cassandra 4.x
- (Optional) Docker Desktop
- (Optional) OpenAI API key for AI endpoint

---

## Setup: Database

1. Start Cassandra locally or connect to a remote cluster.
2. Create keyspace:
    ```sql
    CREATE KEYSPACE IF NOT EXISTS iot_data
    WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1};
    ```
3. Create the table:
    ```bash
    cqlsh -f Database/Create_Crypto_table.cql
    ```

---

## Running: Backend

### 1. HTTP API server

```powershell
cd Backend/crypto/backend
go run .\main.go .\analytics.go [AI.go](http://_vscodecontentref_/0)