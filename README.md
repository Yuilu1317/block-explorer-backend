# Block Explorer Backend (Go)

一个基于 Go 实现的区块链浏览器后端项目，目标是做成**可运行、可测试、可解释、可用于面试展示的工程级后端系统**，而不是简单 demo。

当前阶段聚焦 **Block 查询与同步链路**，已经完成 controller / service / rpc / repo / indexer / mapper / types 分层，并补齐了较完整的单元测试。

---

## 当前状态

### 已完成

- Block 查询接口
- Block RPC 封装
- Block PostgreSQL 持久化
- Block DTO / mapper 映射
- Block indexer 同步决策
- Indexer run-once 调试接口
- 基础自动 indexer runner（当前代码中已启动，后续会改为配置控制）
- RPC / DB 错误映射
- Graceful shutdown 基础逻辑
- 单元测试覆盖：controller / service / repo / mapper / indexer / rpc

### 暂未完成

- finalized / safe block 同步策略
- confirmation depth 配置
- parent hash 连续性校验
- reorg 检测与回滚
- transaction 入库
- address 聚合索引
- indexer retry / backoff / worker pool
- metrics / tracing

---

## 技术栈

- Go
- Gin
- PostgreSQL
- GORM
- Ethereum JSON-RPC
- go-ethereum
- httptest / testing

---

## 项目结构

```bash
block-explorer-backend/
├── api/                    # 路由注册
├── cmd/server/             # 程序入口
├── configs/                # 配置模板，本地配置不提交
├── internal/
│   ├── app/                # 应用装配、依赖初始化、HTTP server 生命周期
│   ├── config/             # YAML 配置加载
│   ├── controller/         # HTTP controller 层
│   ├── db/
│   │   ├── models/         # GORM model
│   │   └── repo/           # Repository / DB 访问层
│   ├── indexer/            # Block indexer 与 runner
│   ├── mapper/             # RPC / Entity / DTO 映射
│   ├── rpc/                # Ethereum JSON-RPC 访问层
│   ├── service/            # 业务编排层
│   ├── types/              # DTO、错误定义、response 结构
│   └── utils/              # 通用工具函数
├── migrations/             # SQL migration 草稿
└── scripts/                # 本地脚本
```

---

## 快速开始

### 1. 准备 PostgreSQL

创建数据库：

```sql
CREATE DATABASE block_explorer;
```

### 2. 准备配置文件

真实配置文件不会提交到 GitHub。请从 example 文件复制一份：

```bash
cp configs/config.example.yaml configs/config.yaml
```

然后编辑：

```bash
configs/config.yaml
```

填写你的本地 PostgreSQL DSN 和 Ethereum RPC URL，例如：

```yaml
server:
  port: "8080"

rpc:
  rpc_url: "https://your-ethereum-rpc-url"
  timeout_seconds: 10

db:
  dsn: "host=127.0.0.1 user=postgres password=your_password dbname=block_explorer port=5432 sslmode=disable TimeZone=Asia/Shanghai"
```

### 3. 安装依赖并启动

```bash
go mod tidy
go run ./cmd/server/main.go
```

健康检查：

```bash
curl http://localhost:8080/health
```

---

## API

### Health

```bash
GET /health
```

### Block

```bash
GET  /block/:number
POST /block/sync/:number
POST /blocks/sync?start=0&end=10
```

### Indexer

```bash
GET  /indexer/status
POST /indexer/run-once
```

示例：

```bash
curl http://localhost:8080/indexer/status | jq
curl -X POST http://localhost:8080/indexer/run-once | jq
```

### Transaction

```bash
GET /tx/:hash
```

### Address

```bash
GET /address/:address
```

### Debug

```bash
GET /debug/db-stats
```

---

## Block 线设计

### 分层职责

- `controller`：只处理 HTTP 参数、状态码、response。
- `service`：负责业务编排，例如 DB-first 查询、RPC fallback、同步流程。
- `rpc`：只负责和 Ethereum JSON-RPC 交互，并把外部错误转换为内部错误。
- `repo`：只负责数据库读写和 DB 错误映射。
- `indexer`：负责判断下一个要同步的 block，并驱动同步。
- `mapper`：负责 RPC block、DB entity、DTO 之间的转换。

### 当前同步模型

当前 indexer 使用 DB 作为进度源：

```text
DB latest block number
        ↓
next = latest + 1
        ↓
和 RPC latest 对比
        ↓
决定是否同步 next block
```

如果 DB 为空，则从 block `0` 开始。

### 后续稳定同步策略

当前版本还没有实现 finalized / safe 同步边界。后续更合理的设计是：

```text
stable_target = finalized block
或 stable_target = latest - confirmation_depth
```

DB 只持久化稳定区块；latest 附近的区块可以通过 RPC 查询，但不默认写入 DB，避免 reorg 带来的数据修正复杂度。

---

## 测试覆盖

当前 block 线已经覆盖：

- `block_controller_test`
- `block_service_test`
- `block_repo_test`
- `block_mapper_test`
- `block_indexer_test`
- `block_rpc_test`
- `rpc_error_test`

运行全部测试：

```bash
go test ./...
```

RPC 测试使用 `httptest.Server` 模拟 Ethereum JSON-RPC，不连接真实节点。

已覆盖的 RPC 场景包括：

- `GetLatestBlockNumber` success
- `GetLatestBlockNumber` JSON-RPC error
- `GetLatestBlockNumber` invalid result
- `GetLatestBlockNumber` HTTP 500
- `GetLatestBlockNumber` invalid JSON
- `GetBlockByNumber` success
- `GetBlockByNumber` result null -> `ErrBlockNotFound`
- `GetBlockByNumber` JSON-RPC error
- `GetBlockByNumber` HTTP 500
- `GetBlockByNumber` invalid JSON
- `mapRPCError` canceled / timeout / unknown / nil

---

## 错误处理原则

项目中避免直接比较外部库返回的 `error`，例如不要使用：

```go
if mapped != err { ... }
```

统一使用：

```go
if errors.Is(err, targetErr) { ... }

if mapped := mapRPCError(err); mapped != nil {
    return mapped
}

return fmt.Errorf("operation context: %w", err)
```

这样可以避免某些外部错误类型不可比较导致 panic，同时保留原始错误上下文。

---

## Roadmap

### Block 线

- [x] Block 查询
- [x] Block DB 持久化
- [x] Block mapper
- [x] Block indexer run-once
- [x] Block RPC 测试
- [x] Indexer runner 配置化
- [x] safe / finalized / latest sync target
- [x] start_block 配置
- [ ] Indexer runner 配置化
- [ ] confirmation depth / finalized target
- [ ] parent hash 连续性校验
- [ ] reorg 检测与恢复

### Transaction / Address 线

- [ ] Transaction 入库
- [ ] Address 聚合
- [ ] Token transfer / log 解析

### 工程能力

- [ ] retry / backoff
- [ ] worker pool 并发同步
- [ ] Redis 缓存
- [ ] metrics / tracing
- [ ] migration 工具化

---

### Indexer Configuration

The indexer is controlled by `configs/config.yaml`:

```yaml
indexer:
  auto_start: false
  interval_seconds: 2
  run_timeout_seconds: 3
  sync_target: safe
  start_block: 0
auto_start: whether to start the background indexer when the server starts.
interval_seconds: interval between automatic indexer runs.
run_timeout_seconds: timeout for each indexer run.
sync_target: the highest block tag the indexer syncs to. Supported values: latest, safe, finalized.
start_block: the first block to sync when the database is empty.

For this MVP, the default sync_target is safe, which balances freshness and stability. Full reorg handling is not implemented yet.