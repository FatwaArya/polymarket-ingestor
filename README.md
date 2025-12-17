# Polymarket Trades Ingestor

A simple real-time trades data ingestor for [Polymarket](https://polymarket.com/) using [QuestDB](https://questdb.io/) as the time-series database.

Built for learning purposes — exploring Go and QuestDB.

## Updated architecture (with Kafka)

```
Polymarket WebSocket → Go app (producer) → Kafka (Redpanda) → Redpanda Connect → QuestDB
```

## Prerequisites
- Docker + Docker Compose
- Go toolchain

## Setup (infrastructure)

1) Start services
```bash
docker compose up -d
```
- QuestDB: http://localhost:9000  
- Redpanda Console (Kafka UI): http://localhost:8080  

2) Verify Redpanda Connect is running
```bash
docker logs redpanda-connect
```

Redpanda Connect will automatically:
- Consume messages from the `polymarket-trades` topic
- Write them to QuestDB table `polymarket_trades`

3) Producer setup
- Producer should publish JSON trades to topic `polymarket-trades` on broker `localhost:19092` (external Kafka listener).

## Environment
Create `.env` with your Polymarket API credentials (used by the Go app):
```env
POLYMARKET_APIKEY=your_api_key
POLYMARKET_SECRET=your_secret
POLYMARKET_PASSPHRASE=your_passphrase
```
