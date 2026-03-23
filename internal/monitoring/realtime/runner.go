package realtime

import (
	"context"
	"fmt"
	"log"
	"time"

	"nexus-chain/internal/monitoring/shared"
	ethutil "nexus-chain/pkg/ethereum"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func (l *EventListener) runSubscriptionLoop(ctx context.Context, sub *shared.EventSubscription) {
	for {
		if err := l.subscribeOnce(ctx, sub); err != nil && ctx.Err() == nil {
			log.Printf(
				"subscription stopped for contract=%s event=%s: %v",
				sub.Contract.Address,
				sub.Event.EventName,
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

func (l *EventListener) subscribeOnce(ctx context.Context, sub *shared.EventSubscription) error {
	client, err := ethclient.DialContext(ctx, sub.WSURL)
	if err != nil {
		return fmt.Errorf("dial websocket rpc: %w", err)
	}
	defer client.Close()

	query := ethutil.NewLogFilterQuery(sub.Contract.Address, sub.ABIEvent.ID.Hex())
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
			if err := shared.ProcessRealtimeLog(ctx, l.db, l.rabbitmqClient, sub, vLog); err != nil {
				log.Printf(
					"failed to handle log for contract=%s event=%s tx=%s log_index=%d: %v",
					sub.Contract.Address,
					sub.Event.EventName,
					vLog.TxHash.Hex(),
					vLog.Index,
					err,
				)
			}
		}
	}
}
