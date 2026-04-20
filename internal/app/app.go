package app

import (
	"block-explorer-backend/api"
	"block-explorer-backend/internal/config"
	"block-explorer-backend/internal/controller"
	"block-explorer-backend/internal/db"
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/db/repo"
	"block-explorer-backend/internal/indexer"
	"block-explorer-backend/internal/rpc"
	"block-explorer-backend/internal/service"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	ethClient, rpcClient, err := rpc.NewEthClient(cfg.Rpc.RPCURL)
	if err != nil {
		return fmt.Errorf("new eth client: %w", err)
	}

	txRPC := rpc.NewTxRPC(ethClient, rpcClient, cfg.Rpc.TimeoutSeconds)
	txService := service.NewTxService(txRPC)
	txController := controller.NewTxController(txService)

	blockRPC := rpc.NewBlockRPC(ethClient, rpcClient, cfg.Rpc.TimeoutSeconds)
	blockRepo := repo.NewBlockRepository(database)
	blockService := service.NewBlockService(blockRPC, blockRepo)
	blockController := controller.NewBlockController(blockService)

	addressRPC := rpc.NewAddressRPC(ethClient, rpcClient, cfg.Rpc.TimeoutSeconds)
	addressService := service.NewAddressService(addressRPC)
	addressController := controller.NewAddressController(addressService)

	debugController := controller.NewDebugController(sqlDB)

	blockIndexer := indexer.NewBlockIndexer(blockRPC, blockRepo, blockService)
	indexerController := controller.NewIndexerController(blockIndexer)

	router := api.NewRouter(
		txController,
		blockController,
		addressController,
		debugController,
		indexerController,
	)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := indexer.NewRunner(blockIndexer, 2*time.Second, 3*time.Second)
	go runner.Start(rootCtx)

	addr := ":" + cfg.Server.Port

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("http server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Println("shutdown signal received")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	log.Println("server exited")
	return nil
}
