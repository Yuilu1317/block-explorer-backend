# block-explorer-backend

A Go-based Ethereum block explorer backend.

The service indexes native Ethereum blocks and transactions, stores indexed data in PostgreSQL, and exposes HTTP APIs for querying blocks, transactions, address transaction history, and indexer status.

## Features

- Ethereum JSON-RPC integration
- Block synchronization by block number
- Block range synchronization
- Automatic block indexing
- Native transaction indexing
- Transaction receipt synchronization
- Transaction execution status tracking
- Address transaction history
- Basic reorg detection
- Layered architecture with controller, service, repository, mapper, RPC, and indexer packages
- Unit tests across RPC, repository, service, mapper, controller, and indexer layers

## Tech Stack

- Go
- Gin
- GORM
- PostgreSQL
- go-ethereum
- Ethereum JSON-RPC

## Project Structure

```text
.
├── api
│   └── router.go
├── cmd
│   └── server
│       └── main.go
├── configs
│   └── config.example.yaml
├── internal
│   ├── app
│   ├── config
│   ├── controller
│   ├── db
│   │   ├── models
│   │   └── repo
│   ├── indexer
│   ├── mapper
│   ├── rpc
│   ├── service
│   ├── types
│   └── utils
├── go.mod
├── go.sum
└── README.md
```

## Architecture

The project follows a layered backend architecture.

```text
HTTP Request
  -> Controller
  -> Service
  -> Repository / RPC
  -> Mapper
  -> DTO Response
```

### Controller Layer

Handles HTTP requests and responses.

Responsibilities:

- Parse path and query parameters
- Call service methods
- Map service errors to HTTP status codes
- Return JSON responses

### Service Layer

Contains business logic.

Responsibilities:

- Block synchronization
- Transaction indexing
- Receipt synchronization
- Reorg and chain continuity checks
- Address transaction query orchestration

### Repository Layer

Handles database access.

Responsibilities:

- Insert and query blocks
- Insert and query transactions
- Update transaction receipt fields
- Query indexed transaction data
- Query address transaction history

### RPC Layer

Wraps Ethereum JSON-RPC calls.

Responsibilities:

- Fetch blocks
- Fetch transactions
- Fetch transaction receipts
- Fetch address information
- Normalize RPC errors

### Indexer Layer

Controls automatic indexing.

Responsibilities:

- Determine the next block to sync
- Run one indexing step
- Run the continuous indexing loop
- Report indexer status

The indexer does not contain block or transaction business logic. It delegates block synchronization to the service layer.

## Data Flow

### Block Synchronization

```text
BlockService.SyncBlockToDB
  -> fetch block from RPC
  -> validate block and parent hash
  -> build transaction models
  -> insert block and transactions in one DB transaction
  -> sync transaction receipts
```

### Receipt Synchronization

```text
TxService.SyncBlockTransactionReceipts
  -> list transactions missing receipt fields
  -> fetch receipt from RPC
  -> validate receipt against transaction hash, block hash, and block number
  -> update receipt_status and receipt_gas_used
```

## Transaction Receipt Semantics

Transaction receipt status is stored as a nullable field.

```text
receipt_status = null  unknown / not synced yet
receipt_status = 0     transaction included but execution failed
receipt_status = 1     transaction included and execution succeeded
```

`receipt_gas_used` is also nullable.

```text
receipt_gas_used = null  unknown / not synced yet
receipt_gas_used > 0     actual gas used by the transaction
```

This distinction is important because `0` is a valid transaction status and must not be treated as missing data.

## API Endpoints

### Blocks

```http
GET /block/:number
POST /sync/block/:number
POST /sync/blocks?start=:start&end=:end
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
POST /indexer/start
POST /indexer/stop
```

## Configuration

Copy the example configuration file:

```bash
cp configs/config.example.yaml configs/config.yaml
```

Update `configs/config.yaml` with local settings.

Example configuration fields may include:

```yaml
server:
  port: 8080

database:
  dsn: "host=localhost user=postgres password=postgres dbname=block_explorer port=5432 sslmode=disable"

ethereum:
  rpc_url: "http://localhost:8545"

indexer:
  start_block: 0
  interval_seconds: 5
```

Do not commit local configuration files containing passwords, private RPC URLs, API keys, or secrets.

## Running Tests

Run all tests:

```bash
go test ./...
```

Run tests for a single package:

```bash
go test ./internal/service/
go test ./internal/mapper/
go test ./internal/controller/
go test ./internal/db/repo/
go test ./internal/rpc/
```

## Development Commands

Tidy Go modules:

```bash
go mod tidy
```

Run the server:

```bash
go run ./cmd/server
```

Check tracked files:

```bash
git ls-files
```

Check untracked and modified files:

```bash
git status --short
```

Review staged changes before commit:

```bash
git diff --cached
```

## Current Scope

The current implementation focuses on native Ethereum data:

- Blocks
- Native transactions
- Transaction receipts
- Address native transaction history
- Basic indexing lifecycle
- Basic reorg detection

## Future Improvements

- Explicit block sync status fields
- Receipt sync retry tracking
- Reorg rollback support
- ERC20 Transfer event parsing
- Token transfer indexing
- Contract metadata indexing
- Structured logging
- Metrics and observability
- SQL migration management
- Integration tests with a real Ethereum node or forked local chain
