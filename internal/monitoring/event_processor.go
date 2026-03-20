package monitoring

import (
	"context"
	"fmt"
	"log"
	"strings"

	"nexus-chain/ent/parsedeventslog"
	ethutil "nexus-chain/pkg/ethereum"

	"github.com/ethereum/go-ethereum/core/types"
)

func (l *RealtimeEventListener) handleLog(ctx context.Context, sub *eventSubscription, vLog types.Log) error {
	if vLog.Removed {
		log.Printf(
			"skip removed log for contract=%s event=%s tx=%s log_index=%d",
			sub.contract.Address,
			sub.event.EventName,
			vLog.TxHash.Hex(),
			vLog.Index,
		)
		return nil
	}

	parsedData, err := ethutil.DecodeLog(sub.abiEvent, vLog)
	if err != nil {
		return fmt.Errorf("decode event log: %w", err)
	}

	parsedData["contract_address"] = sub.contract.Address
	parsedData["event_name"] = sub.event.EventName
	parsedData["event_topic"] = sub.event.EventTopic

	if err := l.db.ParsedEventsLog.Create().
		SetUID(buildUID(vLog.TxHash.Hex(), int64(vLog.Index))).
		SetEventID(sub.event.ID).
		SetBlockNumber(int64(vLog.BlockNumber)).
		SetTxHash(vLog.TxHash.Hex()).
		SetLogIndex(int64(vLog.Index)).
		SetParsedData(parsedData).
		OnConflictColumns(
			parsedeventslog.FieldTxHash,
			parsedeventslog.FieldLogIndex,
		).
		DoNothing().
		Exec(ctx); err != nil {
		return fmt.Errorf("insert parsed event log: %w", err)
	}

	if int64(vLog.BlockNumber) > sub.lastBlock {
		if err := l.db.MonitorEvent.UpdateOneID(sub.event.ID).
			SetLastBlock(int64(vLog.BlockNumber)).
			Exec(ctx); err != nil {
			return fmt.Errorf("update monitor event last_block: %w", err)
		}
		sub.lastBlock = int64(vLog.BlockNumber)
	}

	return nil
}

func buildUID(txHash string, logIndex int64) string {
	return strings.Join([]string{txHash, fmt.Sprintf("%d", logIndex)}, ":")
}
