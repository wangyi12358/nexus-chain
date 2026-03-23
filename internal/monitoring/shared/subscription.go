package shared

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

const ActiveStatus = int8(1)

type EventSubscription struct {
	Contract *ent.MonitorContract
	Event    *ent.MonitorEvent
	ABIEvent abi.Event
}

func LoadRealtimeSubscriptions(ctx context.Context, db *ent.Client) ([]*EventSubscription, error) {
	return loadSubscriptions(ctx, db, func(contract *ent.MonitorContract) bool {
		if strings.TrimSpace(contract.WsURL) == "" {
			log.Printf("skip contract %s because ws_url is empty", contract.Address)
			return false
		}
		return true
	})
}

func LoadScanSubscriptions(ctx context.Context, db *ent.Client) ([]*EventSubscription, error) {
	return loadSubscriptions(ctx, db, func(contract *ent.MonitorContract) bool {
		if strings.TrimSpace(contract.RPCURL) == "" {
			log.Printf("skip contract %s because rpc_url is empty", contract.Address)
			return false
		}
		return true
	})
}

func (s *EventSubscription) Key() string {
	return s.Event.ID.String()
}

func (s *EventSubscription) RealtimeSignature() string {
	return strings.Join([]string{
		s.Contract.ID.String(),
		s.Contract.Address,
		s.Contract.WsURL,
		string(s.Contract.Abi),
		s.Event.EventName,
		s.Event.EventTopic,
	}, "|")
}

func loadSubscriptions(
	ctx context.Context,
	db *ent.Client,
	allowContract func(contract *ent.MonitorContract) bool,
) ([]*EventSubscription, error) {
	contracts, err := db.MonitorContract.Query().
		Where(monitorcontract.StatusEQ(ActiveStatus)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	subscriptions := make([]*EventSubscription, 0)
	for _, contract := range contracts {
		if !allowContract(contract) {
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
			abiEvent, err := ethutil.LookupEvent(parsedABI, eventRow.EventName, eventRow.EventTopic)
			if err != nil {
				log.Printf("skip event %s for contract %s: %v", eventRow.EventName, contract.Address, err)
				continue
			}

			subscriptions = append(subscriptions, &EventSubscription{
				Contract: contract,
				Event:    eventRow,
				ABIEvent: abiEvent,
			})
		}
	}

	return subscriptions, nil
}
