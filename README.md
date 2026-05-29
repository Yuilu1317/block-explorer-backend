# Block Explorer Backend

## Project Overview

block-explorer-backend is a single-chain Ethereum block explorer backend built with Go.

It reads on-chain data through Ethereum JSON-RPC, synchronizes blocks, native transactions, and transaction receipts into PostgreSQL, and provides query APIs for blocks, transactions, and address transaction history.

Besides basic explorer query features, the project also includes automatic block indexing, native transaction indexing, transaction receipt synchronization, reorg detection, transaction conflict validation, and a wallet-facing query boundary.

## Core Features

- Block query API
- Transaction query API
- Address state query API
- Indexed native transaction query API
- Indexed address transaction history API
- Automatic block indexer
- Native transaction indexing
- Transaction receipt synchronization
- Reorg detection
- Wallet-facing query boundary

## Tech Stack

- Go
- Gin
- GORM
- PostgreSQL
- Ethereum JSON-RPC
- go-ethereum

## Architecture

This project follows a layered backend architecture:

```text
Controller -> Service -> Repository / RPC
Indexer    -> Service -> Repository / RPC
Mapper     -> Entity / DTO conversion
```

Main responsibilities:

- **Controller**: handles HTTP requests and responses.
- **Service**: handles business use cases and query boundaries.
- **RPC**: accesses Ethereum nodes through JSON-RPC.
- **Repository**: reads and writes PostgreSQL data.
- **Mapper**: converts RPC models, database entities, and API DTOs.
- **Indexer**: controls background block, transaction, and receipt synchronization.

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/Yuilu1317/block-explorer-backend.git
cd block-explorer-backend
```

### 2. Prepare PostgreSQL

Create a local PostgreSQL database:

```sql
CREATE DATABASE block_explorer;
```

### 3. Create local config

```bash
cp configs/config.example.yaml configs/config.yaml
```

Update `configs/config.yaml` with your PostgreSQL DSN and Ethereum RPC URL.

Do not commit `configs/config.yaml` if it contains local passwords, private RPC URLs, or API keys.

### 4. Run the server

```bash
go run ./cmd/server
```

Health check:

```bash
curl http://localhost:8080/health
```

## API Overview

### Health

```http
GET /health
```

### Blocks

```http
GET /block/:number
POST /block/sync/:number
POST /blocks/sync?start=:start&end=:end
```

### Transactions

```http
GET /tx/:hash
GET /indexed/tx/:hash
```

### Addresses

```http
GET /address/:address
GET /indexed/address/:address/transactions?page=1&page_size=20
```

### Indexer

```http
GET /indexer/status
POST /indexer/run-once
```

## Testing

Run all tests:

```bash
go test ./...
```

The test suite covers controller, service, repository, mapper, RPC, indexer, and domain error behavior.

## Current Limitations

- Single-chain Ethereum indexing only.
- Native ETH transactions only.
- No ERC20 Transfer event indexing yet.
- Reorg conflicts can be detected, but automatic rollback is not implemented yet.
- No distributed indexer lock for multi-instance deployment.
- No Redis cache yet.
- No production-grade metrics, tracing, or alerting yet.
- No advanced RPC retry, backoff, or provider failover yet.

## Roadmap

- Wallet deposit monitoring
- ERC20 Transfer event indexing
- Token transfer query APIs
- Reorg rollback and recovery
- Confirmation depth / finalized block boundary
- Redis cache for hot queries
- Indexer metrics and alerting
- Multi-chain indexing support