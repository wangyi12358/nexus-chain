package main

import (
	"nexus-chain/internal/monitoring"
	"nexus-chain/internal/net"
	"nexus-chain/pkg/config"
	"nexus-chain/pkg/database"

	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(config.New),
		fx.Provide(database.NewEntClient),
		fx.Provide(net.NewHTTPServer),
		fx.Provide(monitoring.NewRealtimeEventListener),
		fx.Invoke(database.RegisterHooks),
		fx.Invoke(net.StartHTTPServer),
	).Run()
}
