# Block Explorer Backend (Go)

一个基于 Go 实现的区块链浏览器后端（工程级项目），用于支持区块 / 交易 / 地址查询。

当前版本为 **RPC 直连版（无数据库）**，后续将逐步演进为带索引器和持久化存储的完整系统。

---

## 🚀 Features（当前能力）

* 查询区块（Block）
* 查询交易（Transaction）
* 查询地址（Address）
* 基于 Ethereum JSON-RPC
* 分层架构（controller / service / rpc）
* 统一错误处理（error mapping）
* DTO / Raw 数据分层
* Controller 层基础测试（httptest + mock）

---

## 🧱 Project Structure

```bash
block-explorer-backend/
├── cmd/server/            # 程序入口
├── internal/
│   ├── app/               # 应用初始化
│   ├── config/            # 配置管理
│   ├── controller/        # HTTP 层（处理请求）
│   ├── service/           # 业务逻辑层
│   ├── rpc/               # 链上 RPC 封装
│   └── types/             # DTO / 数据结构
```

---

## 🔧 Tech Stack

* Go
* gin（HTTP 框架）
* Ethereum JSON-RPC

---

## ▶️ Run

```bash
go mod tidy
go run cmd/server/main.go
```

默认启动后：

```bash
GET /tx/:hash
GET /block/:number
GET /address/:address
```

---

## 🧪 Test

```bash
go test ./...
```

当前测试覆盖：

* Controller 层（基于 httptest + mock service）

---

## 📌 Current Stage

当前为第一阶段：

👉 RPC 直连查询（无缓存 / 无数据库）

---

## 🗺️ Roadmap（下一步）

* [ ] 引入数据库（Block 持久化）
* [ ] Repository 层实现
* [ ] Indexer（区块同步与索引）
* [ ] Transaction / Address 入库
* [ ] 缓存层（提升查询性能）

---

## 💡 Design Notes

* Controller 层只负责 HTTP 处理，不包含业务逻辑
* Service 层负责业务编排
* RPC 层封装链上交互
* DTO 用于隔离链上原始数据结构

---

## 🎯 Goal

该项目目标是实现一个：

👉 可运行 / 可扩展 / 可用于面试的区块链后端服务

而不是简单 demo

---

## 👤 Author

* Go backend learner transitioning into blockchain
* Focus on building real-world backend systems
