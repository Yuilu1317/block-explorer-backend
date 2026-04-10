package app

import (
	"fmt"

	"block-explorer-backend/api"
	"block-explorer-backend/internal/config"
	"block-explorer-backend/internal/controller"
	"block-explorer-backend/internal/rpc"
	"block-explorer-backend/internal/service"
)

func Run() error {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ethClient, err := rpc.NewEthClient(cfg.Rpc.RPCURL)
	if err != nil {
		return fmt.Errorf("new eth client: %w", err)
	}

	txRPC := rpc.NewTxRPC(ethClient, cfg.Rpc.TimeoutSeconds)
	txService := service.NewTxService(txRPC)
	txController := controller.NewTxController(txService)

	blockRPC := rpc.NewBlockRPC(ethClient, cfg.Rpc.TimeoutSeconds)
	blockService := service.NewBlockService(blockRPC)
	blockController := controller.NewBlockController(blockService)

	router := api.NewRouter(
		txController,
		blockController,
	)

	addr := ":" + cfg.Server.Port
	return router.Run(addr)
}
