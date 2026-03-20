# Listen for New Blocks Example

This example demonstrates how to use go-ethereum to listen for new block headers on the Ethereum blockchain.

## Prerequisites

- Go 1.25 or later
- An Ethereum RPC endpoint (e.g., Infura, Alchemy, or a local node)

## Usage

1. Set your RPC URL in `main.go`. For example, use a WebSocket URL from Infura.

2. Run the example:

   ```bash
   go run main.go
   ```

This will start listening for new block headers and log them to the console.

## Notes

- For production, handle errors and reconnections properly.
- You can subscribe to other events like logs, pending transactions, etc., using similar methods.

# Listen for Transfer Events Example

This example demonstrates how to use go-ethereum to listen for Transfer events on the Ethereum blockchain for a specific ERC-20 token contract.

## Prerequisites

- Go 1.25 or later
- An Ethereum RPC endpoint (e.g., Infura, Alchemy, or a local node)

## Usage

1. Set your RPC URL and contract address in `main.go`. For example, use a WebSocket URL from Infura and the address of an ERC-20 token like USDT.

2. Run the example:

   ```bash
   go run main.go
   ```

This will start listening for Transfer events on the specified contract and log the from, to, and value fields.

## Notes

- For production, handle errors, reconnections, and rate limits properly.
- You can modify the function to listen for other events by changing the event signature and ABI.
