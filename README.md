# Polymarket Trades Ingestor

A simple real-time trades data ingestor for [Polymarket](https://polymarket.com/) using [QuestDB](https://questdb.io/) as the time-series database.

Built for learning purposes — exploring Go and QuestDB.

## Setup

1. Start QuestDB:
   ```bash
   docker compose up -d
   ```

2. Create `.env` file:
   ```env
   POLYMARKET_APIKEY=your_api_key
   POLYMARKET_SECRET=your_secret
   POLYMARKET_PASSPHRASE=your_passphrase
   ```

3. Run:
   ```bash
   go run main.go
   ```
