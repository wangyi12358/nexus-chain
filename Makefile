include .env
export

start:
	go run -mod=mod cmd/nexus-chain/main.go

ent-generate:
	go run -mod=mod cmd/ent/generate/main.go