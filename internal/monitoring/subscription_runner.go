package monitoring

import (
	"context"
	"fmt"
	"log"
	"time"

	ethutil "nexus-chain/pkg/ethereum"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

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

	query := ethutil.NewLogFilterQuery(sub.contract.Address, sub.event.EventTopic)
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
