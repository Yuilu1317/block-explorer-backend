package app

import (
	"block-explorer-backend/internal/db"
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/db/repo"
	"fmt"
	"log"

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

	database, err := db.NewDB(cfg)
	if err != nil {
		return err
	}
	if err := database.AutoMigrate(&models.Block{}); err != nil {
		return fmt.Errorf("auto migrate block table failed: %w", err)
	}
	log.Println("database migrated")

	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("get db: %w", err)
	}

	ethClient, err := rpc.NewEthClient(cfg.Rpc.RPCURL)
	if err != nil {
		return fmt.Errorf("new eth client: %w", err)
	}

	txRPC := rpc.NewTxRPC(ethClient, cfg.Rpc.TimeoutSeconds)
	txService := service.NewTxService(txRPC)
	txController := controller.NewTxController(txService)

	blockRPC := rpc.NewBlockRPC(ethClient, cfg.Rpc.TimeoutSeconds)
	blockRepo := repo.NewBlockRepository(database)
	blockService := service.NewBlockService(blockRPC, blockRepo)
	blockController := controller.NewBlockController(blockService)

	addressRPC := rpc.NewAddressRPC(ethClient, cfg.Rpc.TimeoutSeconds)
	addressService := service.NewAddressService(addressRPC)
	addressController := controller.NewAddressController(addressService)

	debugController := controller.NewDebugController(sqlDB)

	router := api.NewRouter(
		txController,
		blockController,
		addressController,
		debugController,
	)

	addr := ":" + cfg.Server.Port
	return router.Run(addr)
}
