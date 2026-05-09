package core

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"nexus-chain/ent"
	ethutil "nexus-chain/pkg/ethereum"
	"nexus-chain/pkg/rabbitmq"

	"github.com/ethereum/go-ethereum/core/types"
)

func ProcessRealtimeLog(ctx context.Context, db *ent.Client, client *rabbitmq.Client, sub *EventSubscription, vLog types.Log) error {
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

	return processLog(ctx, db, client, sub, vLog, "realtime")
}

func ProcessHistoricalLog(ctx context.Context, db *ent.Client, client *rabbitmq.Client, sub *EventSubscription, vLog types.Log) error {
	if vLog.Removed {
		return nil
	}

	return processLog(ctx, db, client, sub, vLog, "scanner")
}

func processLog(
	ctx context.Context,
	db *ent.Client,
	client *rabbitmq.Client,
	sub *EventSubscription,
	vLog types.Log,
	source string,
) error {
	parsedData, err := ethutil.DecodeLog(sub.ABIEvent, vLog)
	if err != nil {
		return fmt.Errorf("decode event log: %w", err)
	}

	parsedData["contract_address"] = sub.Contract.Address
	parsedData["event_name"] = sub.Event.EventName
	parsedData["event_topic"] = sub.ABIEvent.ID.Hex()

	uid := buildUID(vLog.TxHash.Hex(), int64(vLog.Index))
	_, inserted, err := insertParsedEventLog(ctx, db, sub, vLog, uid, parsedData)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	if err := client.PublishEvent(ctx, sub.Event.MqRoutingKey, rabbitmq.EventMessage{
		UID:             uid,
		EventID:         sub.Event.ID.String(),
		ChainID:         sub.Contract.ChainID,
		ContractID:      sub.Contract.ID.String(),
		ContractAddress: sub.Contract.Address,
		EventName:       sub.Event.EventName,
		EventTopic:      sub.ABIEvent.ID.Hex(),
		RoutingKey:      sub.Event.MqRoutingKey,
		BlockNumber:     int64(vLog.BlockNumber),
		TxHash:          vLog.TxHash.Hex(),
		LogIndex:        int64(vLog.Index),
		Source:          source,
		ParsedData:      parsedData,
		PublishedAt:     time.Now(),
	}); err != nil {
		return fmt.Errorf("publish rabbitmq message: %w", err)
	}

	return nil
}

func insertParsedEventLog(
	ctx context.Context,
	db *ent.Client,
	sub *EventSubscription,
	vLog types.Log,
	uid string,
	parsedData map[string]interface{},
) (*ent.ParsedEventsLog, bool, error) {
	entity, err := db.ParsedEventsLog.Create().
		SetUID(uid).
		SetChainID(sub.Contract.ChainID).
		SetEventID(sub.Event.ID).
		SetBlockNumber(int64(vLog.BlockNumber)).
		SetTxHash(vLog.TxHash.Hex()).
		SetLogIndex(int64(vLog.Index)).
		SetParsedData(parsedData).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("insert parsed event log: %w", err)
	}

	return entity, true, nil
}

func buildUID(txHash string, logIndex int64) string {
	return strings.Join([]string{txHash, fmt.Sprintf("%d", logIndex)}, ":")
}
