package realtime

import (
	"context"
	"log"
	"sync"
	"time"

	"nexus-chain/ent"
	"nexus-chain/internal/monitoring/shared"
	"nexus-chain/pkg/config"
	"nexus-chain/pkg/rabbitmq"

	"go.uber.org/fx"
)

const (
	reconnectInterval = 5 * time.Second
	refreshInterval   = time.Minute
)

type EventListener struct {
	db            *ent.Client
	cfg           *config.Config
	publisher     rabbitmq.Publisher
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.Mutex
	subscriptions map[string]*managedSubscription
}

type managedSubscription struct {
	cancel    context.CancelFunc
	signature string
}

func New(db *ent.Client, cfg *config.Config, publisher rabbitmq.Publisher) *EventListener {
	return &EventListener{
		db:            db,
		cfg:           cfg,
		publisher:     publisher,
		subscriptions: make(map[string]*managedSubscription),
	}
}

func StartServer(lc fx.Lifecycle, listener *EventListener) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return listener.start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			listener.stop()
			return nil
		},
	})
}

func (l *EventListener) start(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel

	if err := l.refreshSubscriptions(ctx, runCtx); err != nil {
		cancel()
		return err
	}

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		l.refreshLoop(runCtx)
	}()

	log.Printf("started realtime event listener")
	return nil
}

func (l *EventListener) stop() {
	if l.cancel != nil {
		l.cancel()
	}
	l.wg.Wait()
}

// 每一段时间更新订阅列表，启动新的订阅，停止已删除或禁用的订阅
func (l *EventListener) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshCtx, cancel := context.WithTimeout(context.Background(), refreshInterval)
			if err := l.refreshSubscriptions(refreshCtx, ctx); err != nil {
				log.Printf("refresh subscriptions failed: %v", err)
			}
			cancel()
		}
	}
}

func (l *EventListener) refreshSubscriptions(ctx context.Context, runCtx context.Context) error {
	desiredSubscriptions, err := shared.LoadRealtimeSubscriptions(ctx, l.db, l.cfg)
	if err != nil {
		return err
	}

	desiredByKey := make(map[string]*shared.EventSubscription, len(desiredSubscriptions))
	for _, subscription := range desiredSubscriptions {
		desiredByKey[subscription.Key()] = subscription
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 启动新的或更新的订阅
	for key, desired := range desiredByKey {
		signature := desired.RealtimeSignature()
		current, exists := l.subscriptions[key]
		if exists && current.signature == signature {
			continue
		}

		if exists {
			current.cancel()
			delete(l.subscriptions, key)
		}

		subCtx, cancel := context.WithCancel(runCtx)
		l.subscriptions[key] = &managedSubscription{
			cancel:    cancel,
			signature: signature,
		}

		l.wg.Add(1)
		go func(subscriptionCtx context.Context, sub *shared.EventSubscription) {
			defer l.wg.Done()
			l.runSubscriptionLoop(subscriptionCtx, sub)
		}(subCtx, desired)

		log.Printf("started subscription for contract=%s event=%s", desired.Contract.Address, desired.Event.EventName)
	}

	// 清理已删除或禁用的订阅
	for key, current := range l.subscriptions {
		if _, exists := desiredByKey[key]; exists {
			continue
		}

		current.cancel()
		delete(l.subscriptions, key)
		log.Printf("stopped subscription key=%s because config was removed or disabled", key)
	}

	return nil
}
