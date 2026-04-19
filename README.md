# Block Explorer Backend (Go)

一个基于 Go 实现的区块链浏览器后端（工程级项目），支持区块 / 交易 / 地址查询，并逐步演进为带索引器和持久化存储的完整系统。

当前版本已具备 **数据库持久化 + 最小 Indexer（手动驱动）能力**。

---

## 🚀 Features（当前能力）

### 查询能力（RPC）

* 查询区块（Block）
* 查询交易（Transaction）
* 查询地址（Address）
* 基于 Ethereum JSON-RPC

### 数据存储

* PostgreSQL 持久化（Block）
* GORM 作为 ORM
* 幂等写入（`ON CONFLICT DO NOTHING`）

### Indexer（同步器）

* 单块同步（`POST /block/sync/:number`）
* 区间同步（`POST /blocks/sync`）
* 自动计算同步进度（DB vs RPC）
* 计算下一同步块（next_to_sync）
* **Run Once 调试能力（手动推进同步）**

### 工程能力

* 分层架构（controller / service / rpc / repo）
* 统一错误处理（error mapping）
* DTO / Raw 数据分层
* 可观测同步状态（debug endpoints）

---

## 🧱 Project Structure

```bash
block-explorer-backend/
├── cmd/server/            # 程序入口
├── internal/
│   ├── app/               # 应用初始化 / runtime 管理
│   ├── config/            # 配置管理
│   ├── controller/        # HTTP 层
│   ├── service/           # 业务逻辑层
│   ├── rpc/               # 链上 RPC 封装
│   ├── repo/              # 数据访问层（DB）
│   └── types/             # DTO / 数据结构
```

---

## 🔧 Tech Stack

* Go
* gin（HTTP 框架）
* PostgreSQL
* GORM
* Ethereum JSON-RPC

---

## ▶️ Run

```bash
go mod tidy
go run cmd/server/main.go
```

---

## 🌐 API

### 查询接口

```bash
GET /tx/:hash
GET /block/:number
GET /address/:address
```

---

### 同步接口（Indexer）

#### 单块同步

```bash
POST /block/sync/:number
```

#### 区间同步

```bash
POST /blocks/sync?start=...&end=...
```

---

### Indexer Debug

#### 执行一次同步（Run Once）

```bash
POST /indexer/run-once
```

示例：

```bash
curl -X POST http://localhost:8080/indexer/run-once | jq
```

返回：

```json
{
  "db_latest": 3,
  "rpc_latest": 24912549,
  "next_to_sync": 4,
  "synced": true,
  "synced_block": 4
}
```

---

#### 查看同步状态

```bash
GET /indexer/status
```

---

## 📌 Current Stage

当前阶段：

👉 **Block Indexing MVP（手动驱动）**

已完成：

* Block 持久化
* 幂等写入
* 同步进度计算
* 单轮 indexer 执行（RunOnce）

未完成：

* 自动 indexer loop
* transaction / address 索引
* retry / 并发优化

---

## 🗺️ Roadmap

* [ ] Indexer Loop（自动同步）
* [ ] Transaction 入库
* [ ] Address 聚合
* [ ] Log / Event 解析
* [ ] 缓存层（Redis）
* [ ] 并发同步优化（worker pool）
* [ ] 可观测性（metrics / tracing）

---

## 💡 Design Notes

* Controller 层只负责 HTTP
* Service 层负责业务编排
* Repo 层负责 DB 访问
* RPC 层负责链上交互
* Indexer 采用“DB 作为进度源”的同步模型

---

## 🎯 Goal

该项目目标是实现一个：

👉 **可运行 / 可调试 / 可扩展 / 可用于面试的区块链后端系统**

而不是简单 demo

---

## 👤 Author

* Go backend learner transitioning into blockchain
* Focus on building real-world backend systems

---

## Database

PostgreSQL

### Setup

```sql
CREATE DATABASE block_explorer;
```

配置：

```yaml
configs/config.yaml
```

运行：

```bash
go run ./cmd/server/main.go
```
