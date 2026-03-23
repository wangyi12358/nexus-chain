package shared

import (
	"context"
	"fmt"
	"log"
	"strings"

	"nexus-chain/ent"
	"nexus-chain/ent/parsedeventslog"
	ethutil "nexus-chain/pkg/ethereum"

	"github.com/ethereum/go-ethereum/core/types"
)

func ProcessRealtimeLog(ctx context.Context, db *ent.Client, sub *EventSubscription, vLog types.Log) error {
	if vLog.Removed {
		log.Printf(
			"skip removed log for contract=%s event=%s tx=%s log_index=%d",
			sub.Contract.Address,
			sub.Event.EventName,
			vLog.TxHash.Hex(),
			vLog.Index,
		)
		return nil
	}

	return processLog(ctx, db, sub, vLog)
}

func ProcessHistoricalLog(ctx context.Context, db *ent.Client, sub *EventSubscription, vLog types.Log) error {
	if vLog.Removed {
		return nil
	}

	return processLog(ctx, db, sub, vLog)
}

func processLog(ctx context.Context, db *ent.Client, sub *EventSubscription, vLog types.Log) error {
	parsedData, err := ethutil.DecodeLog(sub.ABIEvent, vLog)
	if err != nil {
		return fmt.Errorf("decode event log: %w", err)
	}

	parsedData["contract_address"] = sub.Contract.Address
	parsedData["event_name"] = sub.Event.EventName
	parsedData["event_topic"] = sub.Event.EventTopic

	if err := db.ParsedEventsLog.Create().
		SetUID(buildUID(vLog.TxHash.Hex(), int64(vLog.Index))).
		SetEventID(sub.Event.ID).
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

	return nil
}

func buildUID(txHash string, logIndex int64) string {
	return strings.Join([]string{txHash, fmt.Sprintf("%d", logIndex)}, ":")
}
