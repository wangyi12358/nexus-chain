# Nexus Chain

`nexus-chain` 是一个基于 Go 的链上事件监听服务。项目使用 `fx` 管理生命周期，使用 `ent` 连接并自动迁移 PostgreSQL，启动后会根据数据库中的监控配置订阅 EVM 合约事件，并将解析后的日志写入数据库。

## 当前能力

- 启动 HTTP 服务
- 连接 PostgreSQL 并自动执行 schema 创建
- 从 `monitor_contracts` 和 `monitor_events` 读取启用中的监听配置
- 通过 WebSocket RPC 实时订阅合约事件
- 解析事件日志并写入 `parsed_events_log`
- 基于 `tx_hash + log_index` 做幂等去重

## 项目结构

```text
.
├── cmd/nexus-chain/          # 应用入口
├── internal/monitoring/      # 实时事件监听逻辑
├── internal/net/             # HTTP 服务启动
├── pkg/config/               # 环境变量配置
├── pkg/database/             # Ent/PostgreSQL 初始化与生命周期管理
├── ent/schema/               # 数据表模型定义
├── api/openapi.yaml          # 当前维护中的 API 草案
├── examples/listen_transfers # go-ethereum 事件监听示例
└── build/package/Dockerfile  # 镜像构建文件
```

## 运行要求

- Go `1.25.1` 或更高版本
- PostgreSQL
- 可访问的 EVM RPC/WS 节点

## 环境变量

项目使用环境变量加载配置，可参考 `.env.example`：

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=myuser
DB_PASSWORD=mysecretpassword
DB_NAME=nexus-chain
DB_SSLMODE=disable

HTTP_PORT=:8080
```

字段说明：

- `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` / `DB_SSLMODE`：PostgreSQL 连接信息
- `HTTP_PORT`：Gin 服务监听地址，当前实现要求带前导冒号，例如 `:8080`

## 本地启动

1. 准备数据库，并创建 `DB_NAME` 对应的库。
2. 复制环境变量模板并按本地环境修改：

```bash
cp .env.example .env
```

3. 启动服务：

```bash
go run ./cmd/nexus-chain
```

服务启动时会自动：

- 初始化 PostgreSQL 连接
- 自动创建/迁移 Ent schema
- 加载数据库中的监听配置
- 为每个启用的事件启动独立订阅协程
- 启动 HTTP 服务

## 数据模型

### `monitor_contracts`

用于定义要监听的合约。

- `chain_id`：链 ID
- `address`：合约地址
- `name`：合约别名
- `abi`：合约 ABI
- `rpc_url`：历史查询使用的 RPC 地址
- `ws_url`：实时订阅使用的 WebSocket 地址
- `status`：是否启用，`1` 为启用

### `monitor_events`

用于定义某个合约下要监听的事件。

- `contract_id`：关联 `monitor_contracts.id`
- `event_name`：事件名，必须能在 ABI 中找到
- `event_topic`：事件 `topic0`
- `mq_routing_key`：预留的消息路由键字段
- `status`：是否启用，`1` 为启用
- `start_block`：首次启动时的起始块高
- `last_block`：最近处理到的块高

### `parsed_events_log`

用于存储解析后的事件日志。

- `event_id`：关联 `monitor_events.id`
- `block_number`：区块高度
- `tx_hash`：交易哈希
- `log_index`：日志索引
- `parsed_data`：解析后的 JSON 数据
- `created_at`：写入时间

## 如何开始监听

当前监听器不会通过 HTTP 接口动态创建配置，而是直接从数据库读取配置。要让服务开始监听，需要先向数据库插入：

1. 一条启用状态的 `monitor_contracts` 记录
2. 一条或多条启用状态的 `monitor_events` 记录

监听器启动时会校验：

- `ws_url` 非空
- `abi` 可正确解析
- `event_name` 能在 ABI 中找到
- `event_topic` 与 ABI 中对应事件的 topic 一致

校验通过后，服务会持续订阅并在断线后自动重连。

## HTTP API 现状

仓库中存在以下与 HTTP 相关的文件：

- `internal/server/server.go`
- `internal/server/server_test.go`
- `api/openapi.yaml`

但当前实际启动入口使用的是 `internal/net/http.go` 中创建的空 Gin 实例，**并没有把 `/ping` 或其他业务路由挂载到运行中的 HTTP 服务上**。因此：

- `api/openapi.yaml` 目前更接近占位草案
- `internal/server/server.go` 中的 `/ping` 仅在测试里被使用
- 现阶段 README 不再将 `/ping` 视为实际可用接口

## 开发命令

运行服务：

```bash
go run ./cmd/nexus-chain
```

重新生成 Ent 代码：

```bash
go run ./cmd/ent/generate
```

运行测试：

```bash
go test ./...
```

## Docker

项目包含镜像构建文件：

```bash
docker build -f build/package/Dockerfile -t nexus-chain .
```

`deployments/docker-compose.yml` 当前仅定义了应用服务映射，未包含 PostgreSQL、环境变量注入或完整部署参数，使用前需要补齐。

## 示例

`examples/listen_transfers` 提供了基于 `go-ethereum` 的独立监听示例，可用于理解底层订阅方式：

```bash
cd examples/listen_transfers
go run main.go
```

更详细说明见 `examples/listen_transfers/README.md`。
