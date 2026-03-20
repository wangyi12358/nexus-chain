package ethereum

import (
	"bytes"
	"fmt"
	"strings"

	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func ParseABI(raw []byte) (abi.ABI, error) {
	parsedABI, err := abi.JSON(bytes.NewReader(raw))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("parse abi: %w", err)
	}
	return parsedABI, nil
}

func LookupEvent(parsedABI abi.ABI, eventName, eventTopic string) (abi.Event, error) {
	abiEvent, ok := parsedABI.Events[eventName]
	if !ok {
		return abi.Event{}, fmt.Errorf("event %s not found in abi", eventName)
	}

	if !strings.EqualFold(abiEvent.ID.Hex(), eventTopic) {
		return abi.Event{}, fmt.Errorf("event topic mismatch: abi=%s db=%s", abiEvent.ID.Hex(), eventTopic)
	}

	return abiEvent, nil
}

func NewLogFilterQuery(address, topic string) goethereum.FilterQuery {
	return goethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(address)},
		Topics:    [][]common.Hash{{common.HexToHash(topic)}},
	}
}

func DecodeLog(event abi.Event, vLog types.Log) (map[string]interface{}, error) {
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
