package scanner

import (
	"context"
	"log"
	"sync"
	"time"

	"nexus-chain/ent"

	"go.uber.org/fx"
)

const (
	scanInterval  = 30 * time.Second
	scanBatchSize = int64(1000)
)

type BlockScanner struct {
	db     *ent.Client
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(lc fx.Lifecycle, db *ent.Client) *BlockScanner {
	scanner := &BlockScanner{db: db}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return scanner.start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			scanner.stop()
			return nil
		},
	})

	return scanner
}

func (s *BlockScanner) start(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	if err := s.scanOnce(ctx); err != nil {
		cancel()
		return err
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.scanLoop(runCtx)
	}()

	log.Printf("started block scanner")
	return nil
}

func (s *BlockScanner) stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

func (s *BlockScanner) scanLoop(ctx context.Context) {
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			scanCtx, cancel := context.WithTimeout(context.Background(), scanInterval)
			if err := s.scanOnce(scanCtx); err != nil {
				log.Printf("scan blocks failed: %v", err)
			}
			cancel()
		}
	}
}

func (s *BlockScanner) scanOnce(ctx context.Context) error {
	targets, err := s.loadScanTargets(ctx)
	if err != nil {
		return err
	}

	for _, target := range targets {
		if err := s.scanTarget(ctx, target); err != nil {
			log.Printf(
				"scan target failed for contract=%s event=%s: %v",
				target.Contract.Address,
				target.Event.EventName,
				err,
			)
		}
	}

	return nil
}
