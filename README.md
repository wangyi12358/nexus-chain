# Nexus Chain

`nexus-chain` 是一个基于 Go 的链上事件监听服务。项目使用 `fx` 管理生命周期，使用 `ent` 连接并自动迁移 PostgreSQL，启动后会根据数据库中的监控配置订阅 EVM 合约事件，并将解析后的日志写入数据库。所有 ID 使用 UUID 生成。

## 当前能力

- 启动 HTTP 服务 (基于 Gin)
- 连接 PostgreSQL 并自动执行 schema 创建
- 从 `monitor_contracts` 和 `monitor_events` 读取启用中的监听配置
- 通过 WebSocket RPC 实时订阅合约事件
- 解析事件日志并写入 `parsed_events_log`
- 基于 `tx_hash + log_index` 做幂等去重
- 支持监听多个事件，包括 Transfer 事件
- 使用 go-ethereum 库监听链上信息
- 集成 RabbitMQ 用于消息队列

## 项目结构

遵循 Standard Go Project Layout：

```text
.
├── cmd/nexus-chain/          # 应用入口
├── internal/
│   ├── monitoring/           # 事件监听逻辑 (realtime, scanner)
│   ├── net/                  # HTTP 服务启动
│   └── server/               # Gin 路由定义
├── pkg/
│   ├── config/               # 环境变量配置 (cleanenv)
│   ├── database/             # Ent/PostgreSQL 初始化与生命周期管理
│   ├── ethereum/             # go-ethereum 集成 (ABI, Filter)
│   ├── middleware/           # Gin 中间件 (CORS, Context)
│   └── rabbitmq/             # RabbitMQ 发布和连接
├── ent/                      # Ent ORM 代码 (schema, client, generated)
├── api/openapi.yaml          # API 规范 (当前仅 /ping)
├── examples/listen_transfers # go-ethereum 事件监听示例
├── build/package/Dockerfile  # 镜像构建文件
├── deployments/docker-compose.yml # 部署配置
├── scripts/                  # 构建和工具脚本
└── test/                     # 额外测试数据
```

## 运行要求

- Go `1.25.1` 或更高版本
- PostgreSQL
- 可访问的 EVM RPC/WS 节点 (如 Infura, Alchemy)
- RabbitMQ (可选，用于消息队列)

## 安装依赖

项目使用 Go Modules 管理依赖。安装依赖：

```bash
go mod download
```

如果需要更新依赖：

```bash
go mod tidy
```

## 环境变量

项目使用环境变量加载配置，可参考 `.env.example`：

```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=myuser
DB_PASSWORD=mysecretpassword
DB_NAME=nexus-chain
DB_SSLMODE=disable

# HTTP 服务配置
HTTP_PORT=:8080

# RabbitMQ 配置 (可选)
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# 其他配置 (如 RPC URLs 在数据库中配置)
```

字段说明：

- `DB_*`：PostgreSQL 连接信息
- `HTTP_PORT`：Gin 服务监听地址，当前实现要求带前导冒号，例如 `:8080`
- `RABBITMQ_URL`：RabbitMQ 连接 URL

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

- `id` (UUID)：主键
- `chain_id`：链 ID
- `address`：合约地址
- `name`：合约别名
- `abi`：合约 ABI (JSON)
- `rpc_url`：历史查询使用的 RPC 地址
- `ws_url`：实时订阅使用的 WebSocket 地址
- `status`：是否启用，`1` 为启用
- `created_at` / `updated_at`：时间戳

### `monitor_events`

用于定义某个合约下要监听的事件。

- `id` (UUID)：主键
- `contract_id` (UUID)：关联 `monitor_contracts.id`
- `event_name`：事件名，必须能在 ABI 中找到
- `event_topic`：事件 `topic0`
- `mq_routing_key`：消息路由键 (用于 RabbitMQ)
- `status`：是否启用，`1` 为启用
- `start_block`：首次启动时的起始块高
- `last_block`：最近处理到的块高
- `created_at` / `updated_at`：时间戳

### `parsed_events_log`

用于存储解析后的事件日志。

- `id` (UUID)：主键
- `event_id` (UUID)：关联 `monitor_events.id`
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

### 监听 Transfer 事件

要监听 `event Transfer(address indexed from, address indexed to, uint256 value)`：

1. 在 `monitor_contracts` 中插入合约信息，包括 ABI。
2. 在 `monitor_events` 中插入事件配置，`event_name` 为 "Transfer"。

### 监听多个事件

是的，可以监听多个事件。为每个要监听的事件在 `monitor_events` 中添加记录，服务会为每个启用的事件启动独立的订阅协程。

### 使用 go-ethereum 监听链上信息

项目使用 `github.com/ethereum/go-ethereum` 库：

- `pkg/ethereum/abi.go`：ABI 解析
- `pkg/ethereum/filter.go`：事件过滤
- `internal/monitoring/realtime/`：实时订阅
- `internal/monitoring/scanner/`：历史扫描

底层通过 WebSocket 连接到 EVM 节点，订阅日志事件。

## HTTP API

当前 API 有限：

- `GET /ping`：健康检查，返回 `{"message": "pong"}`

API 规范见 `api/openapi.yaml`。未来可能扩展更多管理接口。

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

构建二进制：

```bash
go build -o bin/nexus-chain ./cmd/nexus-chain
```

格式化代码：

```bash
gofmt -w .
```

## Docker

构建镜像：

```bash
docker build -f build/package/Dockerfile -t nexus-chain .
```

运行容器 (需要配置环境变量)：

```bash
docker run -p 8080:8080 --env-file .env nexus-chain
```

`deployments/docker-compose.yml` 提供完整部署配置，包括 PostgreSQL 和 RabbitMQ。

## 示例

`examples/listen_transfers` 提供了基于 `go-ethereum` 的独立监听示例，可用于理解底层订阅方式：

```bash
cd examples/listen_transfers
go run main.go
```

更详细说明见 `examples/listen_transfers/README.md`。

## 测试

运行所有测试：

```bash
go test ./...
```

测试覆盖：

- HTTP 服务器 (`internal/server/server_test.go`)
- 配置解析
- 数据库操作
- 事件处理逻辑

## 贡献

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

请确保：

- 代码通过 `go test ./...`
- 遵循 Go 格式化 (`gofmt`)
- 更新相关文档

## 许可证

[MIT License](LICENSE) (如果有许可证文件)

## 联系

如有问题，请开启 Issue 或联系维护者。
