package rpc

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// NewEthClient creates an Ethereum RPC client from the given RPC endpoint.
func NewEthClient(rpcURL string) (*ethclient.Client, *rpc.Client, error) {
	rpcClient, err := rpc.Dial(rpcURL)
	if err != nil {
		return nil, nil, fmt.Errorf("dial rpc: %w", err)
	}
	ethClient := ethclient.NewClient(rpcClient)

	return ethClient, rpcClient, nil
}
