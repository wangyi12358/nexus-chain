# Repository Guidelines

## Project Structure & Module Organization
`cmd/nexus-chain` contains the application entrypoint. Core runtime code lives in `internal/` (`server`, `net`, `monitoring`), while reusable packages such as config, database, Ethereum integration, and middleware live in `pkg/`. Database models and generated ORM code are in `ent/`, with schemas under `ent/schema`. API definitions live in `api/openapi.yaml`, container assets in `build/package` and `deployments/`, and sample integrations in `examples/listen_transfers`.

## Build, Test, and Development Commands
Use the Go toolchain directly; the checked-in `Makefile` currently points to a non-existent `cmd/interface` target.

- `go run cmd/nexus-chain/main.go` starts the HTTP service locally.
- `go test ./...` runs all tests, including `internal/server/server_test.go`.
- `go build -o bin/nexus-chain cmd/nexus-chain/main.go` builds the service binary.
- `make ent-generate` regenerates `ent/` code from `ent/schema` via `cmd/ent/generate/main.go`.
- `docker build -f build/package/Dockerfile -t nexus-chain .` builds the container image.

Copy `.env.example` to `.env` before local runs; config is loaded from environment variables via `cleanenv`.

## Coding Style & Naming Conventions
Follow standard Go formatting: tabs for indentation, `gofmt` on every edited file, and grouped imports. Keep package names short and lowercase (`server`, `database`); exported identifiers use `CamelCase`, unexported helpers use `camelCase`. Treat `ent/` as generated output: edit schemas in `ent/schema` or generator code in `cmd/ent`, then regenerate.

## Testing Guidelines
Place tests next to the code they cover using `*_test.go`. Prefer table-driven tests for handlers, config parsing, and monitoring logic. Use `testing` plus `github.com/stretchr/testify/assert`, which is already in use. Run `go test ./...` before opening a PR; add focused tests for new routes, DB hooks, and event-processing branches.

## Commit & Pull Request Guidelines
The current history uses concise, imperative commit subjects such as `Add initial project structure with Docker support and basic API setup`. Keep commits small and scoped. PRs should describe the behavior change, list any env or schema updates, link the relevant issue, and include request/response samples when API behavior changes.

## Security & Configuration Tips
Do not commit real secrets in `.env`. Keep DB and RPC endpoints configurable through environment variables, and document any new required variables in `.env.example`.
