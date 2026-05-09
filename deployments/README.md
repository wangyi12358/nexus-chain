# Docker Compose 部署指南

## 快速开始

```bash
git clone https://github.com/wangyi12358/nexus-chain.git
cd nexus-chain/deployments
docker-compose up -d
```

## 服务说明

| 服务 | 端口 | 说明 |
|------|------|------|
| nexus-chain | 8080 | 主应用服务 |
| postgres | 5432 | PostgreSQL 数据库 |
| rabbitmq | 5672, 15672 | RabbitMQ 消息队列（管理界面: http://localhost:15672） |

## 环境变量配置

编辑 `deployments/.env` 文件来自定义配置：

```bash
# 应用端口
HTTP_PORT=:8080

# 数据库配置
DB_HOST=postgres
DB_PORT=5432
DB_USER=myuser
DB_PASSWORD=mysecretpassword
DB_NAME=nexus-chain
DB_SSLMODE=disable

# RabbitMQ 配置
RABBITMQ_ENABLED=true
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
RABBITMQ_EXCHANGE=nexus.events
RABBITMQ_EXCHANGE_TYPE=topic
RABBITMQ_DURABLE=true
RABBITMQ_QUEUE=nexus.events.queue
```

## 常用命令

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f nexus-chain

# 停止服务
docker-compose down

# 停止服务并删除数据卷
docker-compose down -v

# 重新构建并启动
docker-compose up -d --build
```

## 数据持久化

- PostgreSQL 数据: `postgres_data` 卷
[- RabbitMQ 数据: `rabbitmq_data` 卷
]()
## 健康检查

- PostgreSQL: 每 5 秒检查一次，最多重试 5 次
- RabbitMQ: 每 10 秒检查一次，最多重试 5 次
- nexus-chain: 依赖 PostgreSQL 和 RabbitMQ 健康检查通过后才启动

## 访问服务

- 应用 API: http://localhost:8080/ping
- RabbitMQ 管理界面: http://localhost:15672 (guest/guest)
- PostgreSQL: localhost:5432
