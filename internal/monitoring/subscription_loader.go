package monitoring

import (
	"context"
	"log"
	"strings"

	"nexus-chain/ent"
	"nexus-chain/ent/monitorcontract"
	"nexus-chain/ent/monitorevent"
	ethutil "nexus-chain/pkg/ethereum"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type eventSubscription struct {
	contract  *ent.MonitorContract
	event     *ent.MonitorEvent
	abiEvent  abi.Event
	lastBlock int64
}

func (l *RealtimeEventListener) loadSubscriptions(ctx context.Context) ([]*eventSubscription, error) {
	contracts, err := l.db.MonitorContract.Query().
		Where(monitorcontract.StatusEQ(activeStatus)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	subscriptions := make([]*eventSubscription, 0)
	for _, contract := range contracts {
		if strings.TrimSpace(contract.WsURL) == "" {
			log.Printf("skip contract %s because ws_url is empty", contract.Address)
			continue
		}

		parsedABI, err := ethutil.ParseABI(contract.Abi)
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
			return nil, err
		}

		for _, eventRow := range events {
			abiEvent, err := ethutil.LookupEvent(parsedABI, eventRow.EventName, eventRow.EventTopic)
			if err != nil {
				log.Printf("skip event %s for contract %s: %v", eventRow.EventName, contract.Address, err)
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

func (s *eventSubscription) key() string {
	return s.event.ID.String()
}

func (s *eventSubscription) signature() string {
	return strings.Join([]string{
		s.contract.ID.String(),
		s.contract.Address,
		s.contract.WsURL,
		string(s.contract.Abi),
		s.event.EventName,
		s.event.EventTopic,
	}, "|")
}
