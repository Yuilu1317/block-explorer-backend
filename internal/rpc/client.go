package rpc

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
)

// NewEthClient creates an Ethereum RPC client from the given RPC endpoint.
func NewEthClient(rpcURL string) (*ethclient.Client, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial ethereum rpc: %w", err)
	}
	return client, nil
}
