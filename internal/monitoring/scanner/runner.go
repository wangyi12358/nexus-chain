package scanner

import (
	"context"
	"fmt"

	"nexus-chain/internal/monitoring/shared"
	ethutil "nexus-chain/pkg/ethereum"

	"github.com/ethereum/go-ethereum/ethclient"
)

func (s *BlockScanner) scanTarget(ctx context.Context, target *shared.EventSubscription) error {
	client, err := ethclient.DialContext(ctx, target.Contract.RPCURL)
	if err != nil {
		return fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()

	latestBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("get latest block: %w", err)
	}

	fromBlock := nextScanStart(target.Event.StartBlock, target.Event.LastBlock)
	toBlock := int64(latestBlock)
	if fromBlock > toBlock {
		return nil
	}

	for start := fromBlock; start <= toBlock; start += scanBatchSize {
		end := start + scanBatchSize - 1
		if end > toBlock {
			end = toBlock
		}

		query := ethutil.NewLogFilterQueryRange(target.Contract.Address, target.Event.EventTopic, start, end)
		logs, err := client.FilterLogs(ctx, query)
		if err != nil {
			return fmt.Errorf("filter logs [%d,%d]: %w", start, end, err)
		}

		for _, vLog := range logs {
			if err := shared.ProcessHistoricalLog(ctx, s.db, target, vLog); err != nil {
				return fmt.Errorf("handle historical log tx=%s index=%d: %w", vLog.TxHash.Hex(), vLog.Index, err)
			}
		}

		if err := s.db.MonitorEvent.UpdateOneID(target.Event.ID).
			SetLastBlock(end).
			Exec(ctx); err != nil {
			return fmt.Errorf("update last_block to %d: %w", end, err)
		}
		target.Event.LastBlock = end
	}

	return nil
}

func nextScanStart(startBlock, lastBlock int64) int64 {
	if lastBlock > 0 {
		return lastBlock + 1
	}
	return startBlock
}
