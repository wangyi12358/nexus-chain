package core

import (
	"context"
	"log"
	"strings"

	"nexus-chain/ent"
	"nexus-chain/ent/chainnode"
	"nexus-chain/ent/monitorcontract"
	"nexus-chain/ent/monitorevent"
	"nexus-chain/ent/monitoreventcursor"
	ethutil "nexus-chain/pkg/ethereum"
	"nexus-chain/pkg/rabbitmq"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const ActiveStatus = int8(1)

type EventSubscription struct {
	Contract *ent.MonitorContract
	Event    *ent.MonitorEvent
	Cursor   *ent.MonitorEventCursor
	ABIEvent abi.Event
	RpcUrl   string
	WsUrl    string
}

func LoadRealtimeSubscriptions(ctx context.Context, db *ent.Client, publisher rabbitmq.Publisher) ([]*EventSubscription, error) {
	nodeURLs, err := loadPreferredNodeURLs(ctx, db, "ws")
	if err != nil {
		return nil, err
	}

	return loadSubscriptions(ctx, db, publisher, nodeURLs, false)
}

func LoadScanSubscriptions(ctx context.Context, db *ent.Client, publisher rabbitmq.Publisher) ([]*EventSubscription, error) {
	nodeURLs, err := loadPreferredNodeURLs(ctx, db, "rpc")
	if err != nil {
		return nil, err
	}

	return loadSubscriptions(ctx, db, publisher, nodeURLs, true)
}

func (s *EventSubscription) Key() string {
	return s.Event.ID.String()
}

func (s *EventSubscription) RealtimeSignature() string {
	return strings.Join([]string{
		s.Contract.ID.String(),
		s.Contract.Address,
		s.WsUrl,
		string(s.Contract.Abi),
		s.Event.EventName,
		s.ABIEvent.ID.Hex(),
	}, "|")
}

func loadSubscriptions(
	ctx context.Context,
	db *ent.Client,
	publisher rabbitmq.Publisher,
	nodeURLs map[int]string,
	withCursor bool,
) ([]*EventSubscription, error) {
	contracts, err := db.MonitorContract.Query().
		Where(monitorcontract.StatusEQ(ActiveStatus)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	cursorByEvent := map[string]*ent.MonitorEventCursor{}
	if withCursor {
		cursors, err := db.MonitorEventCursor.Query().
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, cursor := range cursors {
			cursorByEvent[cursor.EventID.String()] = cursor
		}
	}

	subscriptions := make([]*EventSubscription, 0)
	for _, contract := range contracts {
		nodeURL, ok := nodeURLs[contract.ChainID]
		if !ok || nodeURL == "" {
			log.Printf("skip contract %s because no active node is configured for chain_id=%d", contract.Address, contract.ChainID)
			continue
		}

		parsedABI, err := ethutil.ParseABI(contract.Abi)
		if err != nil {
			log.Printf("skip contract %s because abi is invalid: %v", contract.Address, err)
			continue
		}

		events, err := db.MonitorEvent.Query().
			Where(
				monitorevent.ContractIDEQ(contract.ID),
				monitorevent.StatusEQ(ActiveStatus),
			).
			All(ctx)
		if err != nil {
			return nil, err
		}

		for _, eventRow := range events {
			abiEvent, err := ethutil.LookupEvent(parsedABI, eventRow.EventName)
			if err != nil {
				log.Printf("skip event %s for contract %s: %v", eventRow.EventName, contract.Address, err)
				continue
			}
			if err := publisher.EnsureBinding(ctx, eventRow.MqRoutingKey); err != nil {
				return nil, err
			}

			var cursor *ent.MonitorEventCursor
			if withCursor {
				cursor = cursorByEvent[eventRow.ID.String()]
				if cursor == nil {
					cursor, err = db.MonitorEventCursor.Create().
						SetEventID(eventRow.ID).
						Save(ctx)
					if err != nil {
						if ent.IsConstraintError(err) {
							cursor, err = db.MonitorEventCursor.Query().
								Where(monitoreventcursor.EventIDEQ(eventRow.ID)).
								Only(ctx)
						}
						if err != nil {
							return nil, err
						}
					}
					cursorByEvent[eventRow.ID.String()] = cursor
				}
			}

			subscriptions = append(subscriptions, &EventSubscription{
				Contract: contract,
				Event:    eventRow,
				Cursor:   cursor,
				ABIEvent: abiEvent,
				RpcUrl:   nodeURL,
				WsUrl:    nodeURL,
			})
		}
	}

	return subscriptions, nil
}

func loadPreferredNodeURLs(ctx context.Context, db *ent.Client, nodeType string) (map[int]string, error) {
	nodes, err := db.ChainNode.Query().
		Where(
			chainnode.NodeTypeEQ(nodeType),
			chainnode.StatusEQ(ActiveStatus),
		).
		Order(chainnode.ByPriority()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	nodeURLs := make(map[int]string)

	for _, node := range nodes {
		if _, exists := nodeURLs[node.ChainID]; exists {
			continue
		}
		nodeURLs[node.ChainID] = node.URL
	}

	return nodeURLs, nil
}
