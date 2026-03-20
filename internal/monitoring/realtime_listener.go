package monitoring

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"nexus-chain/ent"
	"nexus-chain/ent/monitorcontract"
	"nexus-chain/ent/monitorevent"
	"nexus-chain/ent/parsedeventslog"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/fx"
)

const (
	activeStatus      = int8(1)
	reconnectInterval = 5 * time.Second
)

type RealtimeEventListener struct {
	db     *ent.Client
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type eventSubscription struct {
	contract  *ent.MonitorContract
	event     *ent.MonitorEvent
	abiEvent  abi.Event
	lastBlock int64
}

func NewRealtimeEventListener(lc fx.Lifecycle, db *ent.Client) *RealtimeEventListener {
	listener := &RealtimeEventListener{db: db}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return listener.start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			listener.stop()
			return nil
		},
	})

	return listener
}

func (l *RealtimeEventListener) start(ctx context.Context) error {
	subscriptions, err := l.loadSubscriptions(ctx)
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel

	for _, subscription := range subscriptions {
		sub := subscription
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			l.runSubscriptionLoop(runCtx, sub)
		}()
	}

	log.Printf("started realtime event listener with %d subscriptions", len(subscriptions))
	return nil
}

func (l *RealtimeEventListener) stop() {
	if l.cancel != nil {
		l.cancel()
	}
	l.wg.Wait()
}

func (l *RealtimeEventListener) loadSubscriptions(ctx context.Context) ([]*eventSubscription, error) {
	contracts, err := l.db.MonitorContract.Query().
		Where(monitorcontract.StatusEQ(activeStatus)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query active monitor contracts: %w", err)
	}

	subscriptions := make([]*eventSubscription, 0)
	for _, contract := range contracts {
		if strings.TrimSpace(contract.WsURL) == "" {
			log.Printf("skip contract %s because ws_url is empty", contract.Address)
			continue
		}

		parsedABI, err := abi.JSON(bytes.NewReader(contract.Abi))
		if err != nil {
			log.Printf("skip contract %s because abi is invalid: %v", contract.Address, err)
			continue
		}

		events, err := l.db.MonitorEvent.Query().
			Where(
				monitorevent.ContractIDEQ(contract.ID),
				monitorevent.StatusEQ(activeStatus),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("query monitor events for contract %s: %w", contract.Address, err)
		}

		for _, eventRow := range events {
			abiEvent, ok := parsedABI.Events[eventRow.EventName]
			if !ok {
				log.Printf("skip event %s for contract %s because it is missing in ABI", eventRow.EventName, contract.Address)
				continue
			}

			if !strings.EqualFold(abiEvent.ID.Hex(), eventRow.EventTopic) {
				log.Printf(
					"skip event %s for contract %s because topic mismatch: abi=%s db=%s",
					eventRow.EventName,
					contract.Address,
					abiEvent.ID.Hex(),
					eventRow.EventTopic,
				)
				continue
			}

			lastBlock := eventRow.LastBlock
			if lastBlock == 0 && eventRow.StartBlock > 0 {
				lastBlock = eventRow.StartBlock - 1
			}

			subscriptions = append(subscriptions, &eventSubscription{
				contract:  contract,
				event:     eventRow,
				abiEvent:  abiEvent,
				lastBlock: lastBlock,
			})
		}
	}

	return subscriptions, nil
}

func (l *RealtimeEventListener) runSubscriptionLoop(ctx context.Context, sub *eventSubscription) {
	for {
		if err := l.subscribeOnce(ctx, sub); err != nil && ctx.Err() == nil {
			log.Printf(
				"subscription stopped for contract=%s event=%s: %v",
				sub.contract.Address,
				sub.event.EventName,
				err,
			)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectInterval):
		}
	}
}

func (l *RealtimeEventListener) subscribeOnce(ctx context.Context, sub *eventSubscription) error {
	client, err := ethclient.DialContext(ctx, sub.contract.WsURL)
	if err != nil {
		return fmt.Errorf("dial websocket rpc: %w", err)
	}
	defer client.Close()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(sub.contract.Address)},
		Topics:    [][]common.Hash{{common.HexToHash(sub.event.EventTopic)}},
	}

	logsCh := make(chan types.Log, 128)
	subscription, err := client.SubscribeFilterLogs(ctx, query, logsCh)
	if err != nil {
		return fmt.Errorf("subscribe logs: %w", err)
	}
	defer subscription.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-subscription.Err():
			if err == nil {
				return fmt.Errorf("subscription closed")
			}
			return err
		case vLog := <-logsCh:
			if err := l.handleLog(ctx, sub, vLog); err != nil {
				log.Printf(
					"failed to handle log for contract=%s event=%s tx=%s log_index=%d: %v",
					sub.contract.Address,
					sub.event.EventName,
					vLog.TxHash.Hex(),
					vLog.Index,
					err,
				)
			}
		}
	}
}

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

	parsedData, err := decodeLog(sub.abiEvent, vLog)
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

func decodeLog(event abi.Event, vLog types.Log) (map[string]interface{}, error) {
	parsed := make(map[string]interface{}, len(event.Inputs))

	if len(vLog.Data) > 0 {
		if err := event.Inputs.NonIndexed().UnpackIntoMap(parsed, vLog.Data); err != nil {
			return nil, err
		}
	}

	topics := vLog.Topics
	if !event.Anonymous && len(topics) > 0 {
		topics = topics[1:]
	}

	indexedInputs := make(abi.Arguments, 0)
	for _, input := range event.Inputs {
		if input.Indexed {
			indexedInputs = append(indexedInputs, input)
		}
	}

	if len(indexedInputs) > 0 {
		if err := abi.ParseTopicsIntoMap(parsed, indexedInputs, topics); err != nil {
			return nil, err
		}
	}

	return parsed, nil
}

func buildUID(txHash string, logIndex int64) string {
	return fmt.Sprintf("%s:%d", txHash, logIndex)
}
