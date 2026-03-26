package scanner

import (
	"context"

	"nexus-chain/internal/monitoring/shared"
)

func (s *BlockScanner) loadScanTargets(ctx context.Context) ([]*shared.EventSubscription, error) {
	return shared.LoadScanSubscriptions(ctx, s.db, s.rabbitmqClient)
}
