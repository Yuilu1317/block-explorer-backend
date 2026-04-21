package indexer

import (
	"block-explorer-backend/internal/types"
	"context"
	"fmt"
	"log"
	"time"
)

type LoopIndexer interface {
	RunIndexerOnce(ctx context.Context) (*types.IndexerOnceResult, error)
}

type Runner struct {
	indexer    LoopIndexer
	interval   time.Duration
	runTimeout time.Duration
}

func NewRunner(indexer LoopIndexer, interval time.Duration, runTimeout time.Duration) *Runner {
	return &Runner{
		indexer:    indexer,
		interval:   interval,
		runTimeout: runTimeout,
	}
}

func (r *Runner) Start(ctx context.Context) {
	log.Println("[indexer-runner] started")

	for {
		select {
		case <-ctx.Done():
			log.Println("[indexer-runner] stopped")
			return
		default:
		}

		runCtx, cancel := context.WithTimeout(ctx, r.runTimeout)
		result, err := r.indexer.RunIndexerOnce(runCtx)
		cancel()

		if err != nil {
			log.Printf("[indexer-runner] error: %v\n", err)
		} else {
			dbLatest := "null"
			if result.DBLatest != nil {
				dbLatest = fmt.Sprintf("%d", *result.DBLatest)
			}
			log.Printf(
				"[indexer-runner] db=%s rpc=%d next=%d synced=%v\n",
				dbLatest,
				result.RPCLatest,
				result.NextToSync,
				result.Synced,
			)
		}

		select {
		case <-ctx.Done():
			log.Println("[indexer-runner] stopped")
			return
		case <-time.After(r.interval):
		}
	}
}
