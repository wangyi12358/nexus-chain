include .env
export

start:
	go run -mod=mod cmd/interface/main.go

ent-generate:
	go run -mod=mod cmd/ent/generate/main.go