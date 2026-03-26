package scanner

import (
	"context"

	"nexus-chain/internal/watcher/core"
)

func (s *BlockScanner) loadScanTargets(ctx context.Context) ([]*core.EventSubscription, error) {
	return core.LoadScanSubscriptions(ctx, s.db, s.rabbitmqClient)
}
