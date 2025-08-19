# Crypto Dashboard

A full-stack, real-time cryptocurrency analytics platform for tracking, analyzing, and reporting on major digital assets. Built for extensibility, automation, and professional reporting.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Setup & Installation](#setup--installation)
  - [Database](#database)
  - [Backend](#backend)
  - [Frontend](#frontend)
- [API Reference](#api-reference)
- [Daily Report Generation](#daily-report-generation)
- [Email Subscription](#email-subscription)
- [Cloud Deployment](#cloud-deployment)
- [Development](#development)
- [Demo](#demo)
- [License](#license)

## Features

- **Live Price Tracking:** Real-time and historical price data for major cryptocurrencies
- **Advanced Analytics:** Volatility, trend, min/max, averages, and top movers
- **AI Query:** Natural language to CQL queries via OpenAI for custom analytics
- **PDF Reporting:** Automated, professional daily PDF reports with charts and AI-generated summaries
- **Email Subscription:** Users can subscribe/unsubscribe to daily reports
- **Modern UI:** Responsive React dashboard with interactive charts and analytics

## Architecture

- **Frontend:** React + TypeScript + Vite ([Frontend/crypto](Frontend/crypto))
- **Backend:** Go HTTP API ([Backend/crypto/backend](Backend/crypto/backend))
- **Database:** Apache Cassandra (time-series tables per coin)
- **Ingestion:** Scheduled fetch from CoinGecko API
- **AI Integration:** OpenAI GPT for analytics summaries and NL queries

## Project Structure

```
Crypto-Dashboard/
├── Backend/
│   └── crypto/
│       └── backend/
│           ├── AI.go
│           ├── analytics.go
│           ├── crypto.go
│           ├── Docker_setup.sh
│           ├── Dockerfile
│           ├── go.mod
│           ├── go.sum
│           ├── main.go
│           ├── Report.go
│           └── Image/
├── Database/
│   ├── Create_Crypto_table.cql
│   ├── Email_subscribers.cql
│   └── Email_Verify_table.cql
├── Documents/
│   ├── Dashboard_demo.mp4
│   ├── Demo_video.txt
│   └── report_example.pdf
├── Frontend/
│   └── crypto/
│       ├── public/
│       ├── src/
│       ├── package.json
│       ├── tsconfig.json
│       └── vite.config.ts
├── .gitignore
└── README.md
```

## Setup & Installation

### Database

1. **Install Apache Cassandra** (locally or via Docker)

2. **Create Keyspace:**
   ```sql
   CREATE KEYSPACE IF NOT EXISTS iot_data
   WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1};
   ```

3. **Create Tables:**
   ```bash
   cqlsh -f Database/Create_Crypto_table.cql
   cqlsh -f Database/Email_subscribers.cql
   cqlsh -f Database/Email_Verify_table.cql
   ```

### Backend

1. **Install Go 1.22+**

2. **Install dependencies:**
   ```bash
   cd Backend/crypto/backend
   go mod tidy
   ```

3. **Set OpenAI API Key:**
   ```bash
   export OPENAI_API_KEY="sk-..."
   # or on Windows:
   # $env:OPENAI_API_KEY = "sk-..."
   ```

4. **Run the API server:**
   ```bash
   go run main.go analytics.go AI.go Report.go
   ```
   - Listens on port 8000
   - Expects Cassandra at 127.0.0.1 and keyspace iot_data

5. **Run the ingestion worker:**
   ```bash
   go run crypto.go
   ```

6. **Alternative: Docker setup:**
   ```bash
   docker build -t crypto-ingestor .
   docker run --rm --name crypto-ingestor --network host crypto-ingestor
   ```

### Frontend

1. **Install Node.js 18+ and npm**

2. **Install dependencies:**
   ```bash
   cd Frontend/crypto
   npm install
   ```

3. **Start development server:**
   ```bash
   npm run dev
   ```
   - Opens at http://localhost:5173
   - Calls backend at http://localhost:8000

4. **Build for production:**
   ```bash
   npm run build
   ```

## API Reference

**Base URL:** `http://localhost:8000`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/latest/{coin_id}` | GET | Latest price for a coin |
| `/history/{coin_id}?minutes={n}` | GET | Price history for last N minutes |
| `/average/{coin_id}?start={t}&end={t}` | GET | Average price in range |
| `/at/{coin_id}?timestamp={t}` | GET | Price at/before timestamp |
| `/range/{coin_id}?start={t}&end={t}` | GET | Min/Max price in range |
| `/coins` | GET | List of available coins |
| `/volatility/{coin_id}?start={t}&end={t}` | GET | Standard deviation and mean price in range |
| `/trend/{coin_id}?start={t}&end={t}` | GET | Trend analysis (regression) |
| `/top-movers?minutes={n}` | GET | Top movers in last N minutes |
| `/ask` | POST | Natural language question → CQL + results (text/plain body) |
| `/subscribe` | POST | Subscribe to daily report (email) |
| `/unsubscribe` | POST | Unsubscribe from daily report (email) |
| `/report` | GET | Download daily PDF report |

## Daily Report Generation

### Automated PDF
`generateDailyReportPDF` creates a multi-page PDF with:
- Cover page
- Daily range metrics
- Top gainers/losers
- Charts and AI-generated summaries

### Charts
Generated using `gonum/plot` and included in the PDF

### AI Summaries
Uses OpenAI GPT for concise, professional analysis of tables and charts

### Email Delivery
Users can subscribe to receive the report via email

## Email Subscription

- **Subscribe:** POST to `/subscribe` with email to receive daily reports
- **Unsubscribe:** POST to `/unsubscribe` with email to stop receiving reports
- **Email logic:** See `Report.go` and `Email_subscribers.cql`

## Cloud Deployment

The application is fully deployed in the cloud for production use:

- **Frontend:** Deployed to [Vercel](https://vercel.com) with automatic deployments from the main branch
- **Backend API:** Deployed to [Render](https://render.com) with auto-scaling and health monitoring
- **Database:** Hosted on [DataStax AstraDB](https://astra.datastax.com) (managed Cassandra service)
- **Cron Jobs:** Data ingestion worker running on AWS EC2 instance with scheduled data fetching

### Cloud Configuration

**Frontend (Vercel):**
- Automatic builds from GitHub repository
- Environment variables configured in Vercel dashboard
- Custom domain support with SSL

**Backend (Render):**
- Connected to GitHub for auto-deployments
- Environment variables set in Render dashboard
- Health check endpoint configured

**Database (AstraDB):**
- Secure cloud-native Cassandra database
- Connection via secure connect bundle
- Automatic backups and scaling

**Cron Worker (AWS EC2):**
- Scheduled data ingestion from CoinGecko API
- Systemd service for automatic startup
- CloudWatch monitoring and logging

## Development

### Frontend

**Key files:**
- Components: `src/components/`
- API: `src/services/api.ts`
- Types: `src/types/index.ts`
- Styles: `src/App.css`

**Scripts:**
- `npm run dev` – Start dev server
- `npm run build` – Build for production
- `npm run preview` – Preview built app
- `npm run lint` – Run ESLint

### Backend

**Key files:**
- API server: `main.go`
- Ingestion: `crypto.go`
- Analytics: `analytics.go`
- AI/NL: `AI.go`
- PDF/Email: `Report.go`
- Docker: See `Dockerfile` and `Docker_setup.sh`

## Demo

- **Video:** [Demo Video](Documents/Dashboard_demo.mp4)
- **Sample Report:** See [report_example.pdf](Documents/report_example.pdf)

## License

This project is for educational and non-commercial use. See individual file headers and dependencies for license details.

For questions or contributions, please open an issue or pull request.
