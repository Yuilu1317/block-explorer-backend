# block-explorer-backend
区块浏览器后端

## 一、目标

1、区块查询
* 按 block number 查
* 按 block hash 查

2、交易查询
* 按 tx hash 查
* 返回交易基础信息 + receipt 基础信息

3、地址查询
* 查 ETH balance
* 查 nonce
* 查最近交易列表

整体功能流:
* 数据查询系统：用户请求 → HTTP API → controller → service → (db / rpc) → 返回数据
* 后台线程：indexer（同步器） → RPC 拉链 → 写入 DB

## 二、概述
>* Web 框架：Gin
>* RPC：自己封装 Ethereum JSON-RPC client
>* 数据库：PostgreSQL 或 MySQL
>* ORM/SQL：先用 database/sql + sqlx 或 GORM 
>* 配置：yaml / env
>* 日志：zap 或 slog
>* HTTP client：标准库 net/http
>* 链节点来源：Infura / Alchemy / 自建以太坊 RPC

## 三、功能模块总览

| 层          | 本质职责            |
| ---------- | --------------- |
| controller | 接 HTTP 请求       |
| service    | 业务逻辑          |
| rpc        | 从链上拿原始数据        |
| db/repo    | 从数据库拿结构化数据      |
| indexer    | 把链数据同步到数据库      |
| types      | 定义接口数据结构        |
| middleware | HTTP增强（日志/错误）   |
