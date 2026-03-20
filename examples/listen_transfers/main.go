package main

import (
	"nexus-chain/pkg/ethereum"

	"github.com/ethereum/go-ethereum/common"
)

func main() {
	// Replace with your Ethereum RPC URL, e.g., "wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID"
	rpcURL := "wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID"
	// Replace with the contract address you want to monitor, e.g., an ERC-20 token contract
	contractAddress := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7") // USDT contract on Ethereum mainnet
	ethereum.ListenForTransferEvents(rpcURL, contractAddress)
}
