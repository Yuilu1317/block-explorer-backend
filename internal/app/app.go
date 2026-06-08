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
	if err := database.AutoMigrate(&models.Block{}, &models.Transaction{}); err != nil {
		return fmt.Errorf("auto migrate database tables failed: %w", err)
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

	baseRPC := rpc.NewBaseRPC(
		ethClient,
		rpcClient,
		time.Duration(cfg.Rpc.TimeoutSeconds)*time.Second,
	)

	txRPC := rpc.NewTxRPC(baseRPC)
	txRepo := repo.NewTransactionRepository(database)
	txService := service.NewTxService(txRPC, txRepo, txRepo)
	txController := controller.NewTxController(txService)

	blockRPC := rpc.NewBlockRPC(baseRPC)
	blockRepo := repo.NewBlockRepository(database)
	// txService implements TransactionReceiptSyncer.
	// BlockService uses it to sync transaction receipts after block + transactions are inserted.
	blockService := service.NewBlockService(blockRPC, blockRepo, txService, txRepo, cfg.Indexer.StartBlock)
	blockController := controller.NewBlockController(blockService, blockService)

	addressRPC := rpc.NewAddressRPC(baseRPC)
	addressService := service.NewAddressService(addressRPC, txRepo)
	addressController := controller.NewAddressController(addressService)

	debugController := controller.NewDebugController(sqlDB)

	blockIndexer := indexer.NewBlockIndexer(
		blockRPC, blockRepo, blockService, cfg.Indexer.SyncTarget, cfg.Indexer.StartBlock)
	indexerController := controller.NewIndexerController(blockIndexer)

	walletService := service.NewWalletInternalService(
		cfg.Rpc.ChainID,
		blockRepo,
		txRepo,
	)
	walletController := controller.NewWalletController(walletService)

	router := api.NewRouter(
		txController,
		blockController,
		addressController,
		debugController,
		indexerController,
		walletController,
	)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.Indexer.AutoStart {
		runner := indexer.NewRunner(
			blockIndexer,
			time.Duration(cfg.Indexer.IntervalSeconds)*time.Second,
			time.Duration(cfg.Indexer.RunTimeoutSeconds)*time.Second,
		)
		go runner.Start(rootCtx)
		log.Println("[indexer-runner] auto start enabled")
	} else {
		log.Println("[indexer-runner] auto start disabled")
	}

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
