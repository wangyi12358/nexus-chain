package scanner

import (
	"context"
	"fmt"
	"time"

	"nexus-chain/internal/monitoring/shared"
	ethutil "nexus-chain/pkg/ethereum"

	"github.com/ethereum/go-ethereum/ethclient"
)

func (s *BlockScanner) scanTarget(ctx context.Context, target *shared.EventSubscription) error {
	client, err := ethclient.DialContext(ctx, target.RPCURL)
	if err != nil {
		return fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()

	latestBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("get latest block: %w", err)
	}

	fromBlock := nextScanStart(target.Cursor.ScanLastBlock, latestBlock)
	toBlock := int64(latestBlock)
	if fromBlock > toBlock {
		return nil
	}

	for start := fromBlock; start <= toBlock; start += scanBatchSize {
		end := start + scanBatchSize - 1
		if end > toBlock {
			end = toBlock
		}

		query := ethutil.NewLogFilterQueryRange(target.Contract.Address, target.ABIEvent.ID.Hex(), start, end)
		logs, err := client.FilterLogs(ctx, query)
		if err != nil {
			return fmt.Errorf("filter logs [%d,%d]: %w", start, end, err)
		}

		for _, vLog := range logs {
			if err := shared.ProcessHistoricalLog(ctx, s.db, s.rabbitmqClient, target, vLog); err != nil {
				return fmt.Errorf("handle historical log tx=%s index=%d: %w", vLog.TxHash.Hex(), vLog.Index, err)
			}
		}

		if err := s.db.MonitorEventCursor.UpdateOneID(target.Cursor.ID).
			SetScanLastBlock(end).
			SetLastScannedAt(time.Now()).
			Exec(ctx); err != nil {
			return fmt.Errorf("update last_block to %d: %w", end, err)
		}
		target.Cursor.ScanLastBlock = end
	}

	return nil
}

func nextScanStart(lastBlock int64, latestBlock uint64) int64 {
	if lastBlock > 0 {
		return lastBlock + 1
	}
	return int64(latestBlock)
}
