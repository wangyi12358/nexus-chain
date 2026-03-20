package ethereum

import (
	"context"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TransferEvent represents the Transfer event
type TransferEvent struct {
	From  common.Address
	To    common.Address
	Value *big.Int
}

// ListenForTransferEvents subscribes to Transfer events for a specific contract
func ListenForTransferEvents(rpcURL string, contractAddress common.Address) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal(err)
	}

	// Transfer event signature
	transferSig := []byte("Transfer(address,address,uint256)")
	transferSigHash := common.BytesToHash(transferSig)

	// Create filter query
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics:    [][]common.Hash{{transferSigHash}},
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		log.Fatal(err)
	}

	// Parse the event ABI
	transferAbi := `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	parsedAbi, err := abi.JSON(strings.NewReader(transferAbi))
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case vLog := <-logs:
			var transfer TransferEvent
			err := parsedAbi.UnpackIntoInterface(&transfer, "Transfer", vLog.Data)
			if err != nil {
				log.Printf("Failed to unpack log: %v", err)
				continue
			}
			transfer.From = common.HexToAddress(vLog.Topics[1].Hex())
			transfer.To = common.HexToAddress(vLog.Topics[2].Hex())
			log.Printf("Transfer: from %s to %s value %s", transfer.From.Hex(), transfer.To.Hex(), transfer.Value.String())
		}
	}
}
