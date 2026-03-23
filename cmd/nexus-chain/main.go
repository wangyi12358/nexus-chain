package main

import (
	"nexus-chain/internal/monitoring/realtime"
	"nexus-chain/internal/monitoring/scanner"
	"nexus-chain/internal/net"
	"nexus-chain/pkg/config"
	"nexus-chain/pkg/database"
	"nexus-chain/pkg/rabbitmq"

	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(config.New),
		fx.Provide(database.NewEntClient),
		fx.Provide(rabbitmq.New),
		fx.Provide(net.NewHTTPServer),
		fx.Provide(scanner.New),
		fx.Provide(realtime.New),
		fx.Invoke(realtime.StartServer),
		fx.Invoke(scanner.StartServer),
		fx.Invoke(database.RegisterHooks),
		fx.Invoke(net.StartHTTPServer),
	).Run()
}
