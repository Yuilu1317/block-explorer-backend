# Block Explorer Backend

一个用 Go 实现的区块链浏览器后端项目。
当前项目处于 **Block 同步链路完成、Transaction / Address 索引线待扩展** 阶段。

---

## 1. 项目目标

这个项目主要解决的问题是：

> 把 Ethereum JSON-RPC 返回的链上数据，转换为后端系统可以稳定查询、持久化和继续处理的本地数据。

在真实业务中，类似能力会被用于：

- 区块链浏览器的数据查询
- 钱包系统的充值监听
- 交易所的入账确认
- 地址交易历史查询
- 链上数据平台的数据索引
- 后续 token transfer / event log 解析

当前阶段重点不是覆盖所有链上数据，而是先把 **Block 同步、错误处理、indexer 推进、reorg 风险检测** 这条主链路做扎实。

---

## 2. 技术栈

- Go
- Gin
- PostgreSQL
- GORM
- Ethereum JSON-RPC
- go-ethereum
- testing / httptest
- SQLite in-memory tests for repository layer

---

## 3. 当前完成状态

### 已完成：Block 线

- Block 查询接口
- Block 单块同步
- Block range 手动同步
- Block PostgreSQL 持久化
- Block DTO / mapper 映射
- Ethereum JSON-RPC block 封装
- Indexer run-once 同步
- Indexer runner 自动循环
- Indexer 配置化：
  - `auto_start`
  - `interval_seconds`
  - `run_timeout_seconds`
  - `sync_target`
  - `start_block`
- 支持同步目标：
  - `latest`
  - `safe`
  - `finalized`
- 默认使用 `safe` 作为同步边界
- DB 为空时从 `start_block` 开始同步
- DB 有数据时从 `db_latest + 1` 继续同步
- 同高度相同 hash 的幂等同步
- 同高度不同 hash 的 reorg 检测
- parent hash 连续性检测
- range sync 遇到链安全错误时停止
- reorg / chain discontinuity 映射为 HTTP 409
- controller / service / repo / mapper / rpc / indexer / types 单元测试

### 已有但未完成索引化：Transaction / Address 线

当前项目已有：

- `GET /tx/:hash`
- `GET /address/:address`

但它们目前主要是 **RPC 查询接口**，还不是完整的本地索引能力。

当前还没有完成：

- transaction 表结构设计与持久化
- block 同步时同步 transactions
- transaction by hash 本地查询
- address 相关交易查询
- address 聚合统计
- token transfer / log 解析

---

## 4. 项目结构

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

## 5. 快速开始

### 5.1 准备 PostgreSQL

创建数据库：

```sql
CREATE DATABASE block_explorer;
```

### 5.2 准备配置文件

真实配置文件不提交到 GitHub。请复制 example 配置：

```bash
cp configs/config.example.yaml configs/config.yaml
```

示例配置：

```yaml
server:
  port: "8080"

rpc:
  rpc_url: "https://ethereum-rpc.publicnode.com"
  timeout_seconds: 10

db:
  dsn: "host=127.0.0.1 user=postgres password=your_password dbname=block_explorer port=5432 sslmode=disable TimeZone=Asia/Shanghai"

indexer:
  auto_start: false
  interval_seconds: 2
  run_timeout_seconds: 3
  sync_target: safe
  start_block: 0
```

### 5.3 启动服务

```bash
go mod tidy
go run ./cmd/server/main.go
```

health：

```bash
curl http://localhost:8080/health
```

---

## 6. API

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

示例：

```bash
curl http://localhost:8080/block/100 | jq
curl -X POST http://localhost:8080/block/sync/100 | jq
curl -X POST "http://localhost:8080/blocks/sync?start=100&end=110" | jq
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

### Transaction 当前 RPC 查询

```bash
GET /tx/:hash
```

说明：当前 transaction 接口主要通过 RPC 查询交易详情，尚未完成本地 transaction 索引和入库。

### Address 当前 RPC 查询

```bash
GET /address/:address
```

说明：当前 address 接口主要通过 RPC 查询余额、nonce、code，尚未完成本地地址交易聚合。

### Debug

```bash
GET /debug/db-stats
```

---

## 7. Block 同步设计

### 7.1 分层职责

项目当前采用以下分层：

- `controller`：处理 HTTP 参数、状态码、response。
- `service`：负责业务编排，例如 DB-first 查询、RPC fallback、同步检测、range sync 流程。
- `rpc`：负责 Ethereum JSON-RPC 调用，并将外部错误转换为内部错误。
- `repo`：负责数据库读写和 DB 错误映射。
- `indexer`：负责判断下一个要同步的 block，并驱动同步。
- `mapper`：负责 RPC block、DB entity、DTO 之间的转换。
- `types`：定义 DTO、response、业务错误。

### 7.2 查询逻辑

Block 查询采用 DB-first 逻辑：

```text
1. 先查本地 DB
2. 如果 DB 命中，直接返回 DB 数据
3. 如果 DB 未命中，再通过 RPC 查询链上 block
4. RPC 查询结果只用于响应，不一定写入 DB
```

这样可以减少重复 RPC 调用，同时保留查询未同步区块的能力。

### 7.3 单块同步逻辑

单块同步流程：

```text
1. 从 RPC 拉取 block N
2. 转换为本地 block model
3. 查询 DB 是否已有 block N
4. 如果已有 block N：
   - hash 相同：认为是幂等同步，直接 return nil
   - hash 不同：返回 ErrReorgDetected
5. 如果没有 block N：
   - 查询 DB 是否有 block N-1
   - 如果 parent block 存在，则校验 RPC block N.parent_hash 是否等于 DB block N-1.hash
   - 如果不一致，返回 ErrChainDiscontinuity
   - 如果 parent block 不存在，允许把当前 block 作为本地 start_block 写入
6. 通过检测后写入 DB
```

### 7.4 为什么 parent block 不存在时允许同步

这是为了支持 `start_block`。

例如配置：

```yaml
indexer:
  start_block: 1000000
```

当 DB 为空时，本地没有 block `999999`，但系统仍然应该允许从 block `1000000` 开始建立本地索引。

因此当前规则是：

```text
如果 parent block 存在，则必须检查 parent_hash 连续性。
如果 parent block 不存在，则允许当前 block 作为本地索引边界。
```

这不是完整节点逻辑，而是区块链浏览器后端的索引器边界设计。

### 7.5 Reorg / discontinuity 处理边界

当前阶段只做检测，不做自动恢复。

已经实现：

- 同高度不同 hash：返回 `ErrReorgDetected`
- parent hash 不连续：返回 `ErrChainDiscontinuity`
- 发现链安全错误时，不写入当前 block
- range sync 遇到链安全错误时停止
- HTTP 层返回 409 Conflict

暂未实现：

- 自动 rollback
- 自动寻找共同祖先
- reorg recovery
- transaction/address 的 reorg 修正

这是有意的工程取舍：当前阶段先保证 **发现风险并阻止错误数据继续写入**，而不是一开始就实现复杂恢复逻辑。

---

## 8. Indexer 设计

### 8.1 同步进度

Indexer 使用 DB 作为同步进度源：

```text
DB 有数据：next = db_latest + 1
DB 为空：next = start_block
```

然后获取 RPC 目标高度：

```text
rpc_target = block number by sync_target
```

其中 `sync_target` 支持：

- `latest`
- `safe`
- `finalized`

如果：

```text
next <= rpc_target
```

则执行一次同步。

### 8.2 为什么默认使用 safe

`latest` 最新，但更容易受到短期 reorg 影响。

`finalized` 最稳定，但延迟更高。

`safe` 在新鲜度和稳定性之间更平衡，因此当前 MVP 默认使用：

```yaml
sync_target: safe
```

### 8.3 Runner 行为

Runner 会按照配置间隔循环调用 `RunIndexerOnce`。

单次失败不会停止 runner。原因是：

```text
DBLatest 只有在成功写入后才会推进。
如果某个 block 同步失败，下一轮仍然会从当前 DB latest 的下一个 block 重试。
失败 block 不会被跳过。
```

---

## 9. 错误处理设计

### 9.1 错误包装原则

项目中使用 `%w` 包装错误，保留错误链：

```go
return fmt.Errorf("sync block %d: %w", number, err)
```

上层使用：

```go
errors.Is(err, types.ErrReorgDetected)
```

来识别底层业务错误。

不要用字符串判断错误，例如：

```go
strings.Contains(err.Error(), "reorg")
```

### 9.2 HTTP error mapping

统一通过 `types.WriteError` 和 `types.MapError` 映射错误：

| Error | HTTP Status | Message |
| --- | ---: | --- |
| `ErrInvalidBlockNumber` | 400 | `invalid block number` |
| `ErrBlockNotFound` | 404 | `block not found` |
| `ErrInvalidBlockRange` | 400 | `invalid block range` |
| `ErrBlockRangeTooLarge` | 400 | `block range too large` |
| `ErrReorgDetected` | 409 | `reorg detected` |
| `ErrChainDiscontinuity` | 409 | `chain discontinuity detected` |
| `ErrRPCTimeout` / `ErrDBTimeout` | 504 | `upstream timeout` |
| `ErrRequestCanceled` | 408 | `request canceled` |
| Unknown error | 500 | `internal server error` |

### 9.3 Range sync 错误策略

Range sync 区分两类错误：

```text
普通错误：记录 failed block，继续同步后面的 block。
链安全错误：记录 failed block，立即停止并返回 error。
```

链安全错误包括：

- `ErrReorgDetected`
- `ErrChainDiscontinuity`

原因是这类错误表示本地链状态和 RPC 链状态已经发生冲突，继续同步后续 block 可能扩大数据污染。

---

## 10. 测试

运行全部测试：

```bash
go test ./...
```

当前测试覆盖：

- controller 层：参数解析、成功响应、错误响应
- service 层：DB-first 查询、RPC fallback、单块同步、range sync、reorg/discontinuity 行为
- repo 层：insert、duplicate ignore、latest block、block by number
- mapper 层：RPC block / entity / DTO 映射
- rpc 层：JSON-RPC success / error / null result / invalid JSON / HTTP 500 / timeout mapping
- indexer 层：next block 决策、run once、DB empty、RPC ahead、no new block、错误传递
- types 层：error mapping，尤其是 wrapped error 下的 HTTP 409 映射

重点测试场景：

```text
same number + same hash
=> idempotent sync, no insert

same number + different hash
=> ErrReorgDetected, no insert

block N missing + parent exists + parent hash matches
=> insert block N

block N missing + parent exists + parent hash mismatch
=> ErrChainDiscontinuity, no insert

block N missing + parent missing
=> allow sync from local start_block

range sync + reorg/discontinuity
=> stop range sync and return error
```

---

## 11. 当前工程边界

当前项目还不是完整区块链浏览器。

已经比较完整的是：

```text
Block 查询 + Block 同步 + Block indexer + 基础链安全检测
```

还没有完成的是：

```text
Transaction 本地索引
Address 本地聚合
Token transfer / event log
完整 reorg recovery
生产级 migration
缓存、监控、并发同步
```

---

## 12. 后续 Roadmap

### 12.1 Transaction 线

transaction 存储层：

- transaction model
- transaction repository
- batch insert transactions
- query transaction by hash from DB
- block sync 时同步 transactions
- transaction mapper
- transaction service / controller tests

关键工程点：

- Ethereum transaction 的 `from` 不是普通字段，需要通过 signer 从签名恢复
- 需要考虑 chain_id
- EIP-1559 交易不只有 gas_price，还涉及 fee cap / tip cap
- block 和 transactions 最好在同一个 DB transaction 中写入
- reorg 后 transaction/address 数据也需要修正

### 12.2 Address 线

address 查询：

- 根据 address 查询相关 transactions
- 支持 from / to 查询
- 支持分页
- 地址交易数量统计
- 地址最新活动 block

注意：不靠普通 transaction 简单计算地址余额：

- gas fee
- internal transfer
- contract call 内部转账
- ERC20 transfer
- reorg 修正

最小版本可以通过 RPC 查询 ETH balance，本地 DB 先负责交易历史和统计。

### 12.3 Reorg recovery

当前只检测，不恢复。后续可以独立设计：

- 检测 reorg 深度
- 回退本地 block
- 回滚 transaction/address 派生数据
- 寻找共同祖先
- 从共同祖先之后重新同步

### 12.4 工程增强

- migration 工具化
- retry / backoff
- structured logging
- metrics / tracing
- Redis cache
- worker pool 并发同步
- graceful shutdown 更细化
- integration tests with PostgreSQL container

---

