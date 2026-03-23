package database

import (
	"context"
	"fmt"
	"time"

	"nexus-chain/ent"
	"nexus-chain/ent/migrate"

	"go.uber.org/fx"
)

func RegisterHooks(lc fx.Lifecycle, client *ent.Client) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			migrateCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			if err := client.Schema.Create(
				migrateCtx,
				migrate.WithForeignKeys(true),
			); err != nil {
				return fmt.Errorf("auto migrate schema on startup: %w", err)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return client.Close()
		},
	})
}
