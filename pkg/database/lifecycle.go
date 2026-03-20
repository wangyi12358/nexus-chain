package database

import (
	"context"

	"nexus-chain/ent"

	"go.uber.org/fx"
)

func RegisterHooks(lc fx.Lifecycle, client *ent.Client) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return client.Close()
		},
	})
}
