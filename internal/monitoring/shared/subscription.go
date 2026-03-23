package shared

import (
	"context"
	"log"
	"strings"

	"nexus-chain/ent"
	"nexus-chain/ent/monitorcontract"
	"nexus-chain/ent/monitorevent"
	"nexus-chain/pkg/config"
	ethutil "nexus-chain/pkg/ethereum"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const ActiveStatus = int8(1)

type EventSubscription struct {
	Contract *ent.MonitorContract
	Event    *ent.MonitorEvent
	ABIEvent abi.Event
	RPCURL   string
	WSURL    string
}

func LoadRealtimeSubscriptions(ctx context.Context, db *ent.Client, cfg *config.Config) ([]*EventSubscription, error) {
	wsURL, err := cfg.WsUrl()
	if err != nil {
		return nil, err
	}

	return loadSubscriptions(ctx, db, func(contract *ent.MonitorContract) (string, string, bool, error) {
		return "", wsURL, true, nil
	})
}

func LoadScanSubscriptions(ctx context.Context, db *ent.Client, cfg *config.Config) ([]*EventSubscription, error) {
	rpcURL, err := cfg.RpcUrl()
	if err != nil {
		return nil, err
	}

	return loadSubscriptions(ctx, db, func(contract *ent.MonitorContract) (string, string, bool, error) {
		return rpcURL, "", true, nil
	})
}

func (s *EventSubscription) Key() string {
	return s.Event.ID.String()
}

func (s *EventSubscription) RealtimeSignature() string {
	return strings.Join([]string{
		s.Contract.ID.String(),
		s.Contract.Address,
		s.WSURL,
		string(s.Contract.Abi),
		s.Event.EventName,
		s.ABIEvent.ID.Hex(),
	}, "|")
}

func loadSubscriptions(
	ctx context.Context,
	db *ent.Client,
	resolveNodeURLs func(contract *ent.MonitorContract) (rpcURL, wsURL string, ok bool, err error),
) ([]*EventSubscription, error) {
	contracts, err := db.MonitorContract.Query().
		Where(monitorcontract.StatusEQ(ActiveStatus)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	subscriptions := make([]*EventSubscription, 0)
	for _, contract := range contracts {
		rpcURL, wsURL, ok, err := resolveNodeURLs(contract)
		if err != nil {
			return nil, err
		}
		if !ok {
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

			subscriptions = append(subscriptions, &EventSubscription{
				Contract: contract,
				Event:    eventRow,
				ABIEvent: abiEvent,
				RPCURL:   rpcURL,
				WSURL:    wsURL,
			})
		}
	}

	return subscriptions, nil
}
