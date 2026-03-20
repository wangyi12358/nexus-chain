package main

import (
	"context"
	"log"

	ethutil "nexus-chain/pkg/ethereum"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// Replace with your Ethereum RPC URL, e.g., "wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID"
	rpcURL := "wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID"
	// Replace with the contract address you want to monitor, e.g., an ERC-20 token contract
	contractAddress := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7") // USDT contract on Ethereum mainnet
	transferABI := `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`

	parsedABI, err := ethutil.ParseABI([]byte(transferABI))
	if err != nil {
		log.Fatal(err)
	}

	transferEvent, err := ethutil.LookupEvent(parsedABI, "Transfer", parsedABI.Events["Transfer"].ID.Hex())
	if err != nil {
		log.Fatal(err)
	}

	client, err := ethclient.DialContext(context.Background(), rpcURL)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	query := ethutil.NewLogFilterQuery(contractAddress.Hex(), transferEvent.ID.Hex())
	logsCh := make(chan types.Log, 32)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logsCh)
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case vLog := <-logsCh:
			parsed, err := ethutil.DecodeLog(transferEvent, vLog)
			if err != nil {
				log.Printf("decode failed: %v", err)
				continue
			}
			log.Printf("Transfer contract=%s parsed=%+v", contractAddress.Hex(), parsed)
		}
	}
}
