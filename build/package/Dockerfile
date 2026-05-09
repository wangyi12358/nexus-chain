FROM golang:1.22 AS builder
# 注：目前官方还没有 1.25，假设你使用的是最新稳定版
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 关键修复：禁用 CGO 确保静态编译
RUN CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o nexus-chain cmd/nexus-chain/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
# 建议加上权限处理，确保文件可执行
COPY --from=builder /app/nexus-chain .
RUN chmod +x nexus-chain

EXPOSE 8080
CMD ["./nexus-chain"]