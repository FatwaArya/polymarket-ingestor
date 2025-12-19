# Polymarket Trades Ingestor

A simple real-time trades data ingestor for [Polymarket](https://polymarket.com/) using [QuestDB](https://questdb.io/) as the time-series database.

Built for learning purposes — exploring Go and QuestDB.

## Updated architecture (with Kafka)

```
Polymarket WebSocket → Go app (producer) → Kafka (Redpanda) → Redpanda Connect → QuestDB
```

## Codebase architecture & patterns

At a high level, the Go app is a small streaming service that:

- **Connects to Polymarket WebSocket** (`internal`):
  - `internal.WebSocketClient` owns the WebSocket connection and subscription lifecycle.
  - It knows how to:
    - connect to `wss://ws-live-data.polymarket.com`
    - subscribe/unsubscribe to topics (e.g. `activity` → `trades`)
    - keep the connection alive via ping/pong
    - push raw messages to a callback for further handling

- **Parses domain data** (`utils`):
  - `utils.IncomingMessage` and `utils.ActivityTradePayload` are DTOs for the WebSocket payloads.
  - `utils.ParseActivityTrade`:
    - filters for `activity` + `trades` messages
    - unmarshals JSON into a strongly-typed `ActivityTradePayload`
    - returns a sentinel `ErrSkipMessage` for non-trade or non-JSON messages (so the caller can cheaply ignore noise).

- **Produces to Kafka** (`internal/kafka`):
  - `kafka.Producer` wraps the `franz-go` client:
    - `NewProducer` configures the client for a given broker list + topic.
    - `ProduceTrade` maps `ActivityTradePayload` → `TradeMessage` (a flat JSON-friendly struct) and sends it to Kafka.
    - Uses the transaction hash as the Kafka key when available so related records land in the same partition.
  - `kafka.Consumer` is a simple, currently-unused wrapper that demonstrates how to read from the same topic in another service.

- **(Optional) Writes directly to QuestDB** (`internal/questdb.ingest.go`):
  - `TradeWriter` wraps a QuestDB Line Sender (TCP or HTTP).
  - Exposes `Write`, `WriteBatch`, `Flush`, and `Close` to stream trades into the `polymarket_trades` table.
  - This is an alternative to the Redpanda Connect path, and can be used by a separate worker process.

- **Configuration** (`config`):
  - `config.Config` is a simple configuration struct populated from environment variables (with `.env` support via `godotenv`).
  - The package exposes a global `config.AppConfig` initialized in `init()` and sets the global Gin mode.
  - Key settings: HTTP port, Kafka brokers/topic, QuestDB host/port, and Polymarket API credentials.

- **Application entrypoint** (`main.go`):
  - Reads configuration from `config.AppConfig`.
  - Builds subscriptions for Polymarket (currently `activity` → `trades`).
  - Creates a Kafka `Producer` and a `WebSocketClient`, wiring them together via a callback:
    1. WebSocket receives raw bytes.
    2. `utils.ParseActivityTrade` turns them into `ActivityTradePayload`.
    3. The producer publishes JSON trade messages to Kafka.
  - Exposes a small HTTP health/`/ping` endpoint and starts:
    - the Gin HTTP server
    - the WebSocket client loop
    - a pprof server on `:6060` for runtime inspection
  - Handles graceful shutdown via OS signals and closes the WebSocket client and Kafka producer.

### Current architectural pattern

- **Pattern style**: a small, layered streaming service:
  - **Transport / I/O layer**: WebSocket client, Kafka client, QuestDB client.
  - **Domain / parsing layer**: DTOs + parsing logic in `utils`.
  - **Application wiring**: `main.go` composes everything and defines the data flow.
- **Coupling**:
  - The WebSocket client is kept generic (it only knows about subscriptions and a byte callback).
  - Domain parsing and Kafka production are separated into their own packages so you can:
    - reuse them in other services
    - test them in isolation
- **Extension points**:
  - Add more subscriptions (e.g. comments, `clob_user`) using `internal.Subscription` helpers.
  - Add new sinks: another Kafka topic, a different DB writer, or additional workers consuming from Kafka.

This documentation describes the **current** shape of the codebase and can be used as a reference when evolving toward a stricter “clean” or hexagonal architecture later.

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
