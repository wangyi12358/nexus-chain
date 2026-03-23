package ethereum

import (
	"math/big"

	goethereum "github.com/ethereum/go-ethereum"
)

func NewLogFilterQueryRange(address, topic string, fromBlock, toBlock int64) goethereum.FilterQuery {
	query := NewLogFilterQuery(address, topic)
	query.FromBlock = big.NewInt(fromBlock)
	query.ToBlock = big.NewInt(toBlock)
	return query
}
