package indexer

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

type IndexerRunner interface {
	RunIndexerOnce(ctx context.Context) (*types.IndexerOnceResult, error)
}

type Runner struct {
	indexerRunner IndexerRunner
	interval      time.Duration
	runTimeout    time.Duration
}

func NewRunner(indexerRunner IndexerRunner, interval time.Duration, runTimeout time.Duration) *Runner {
	return &Runner{
		indexerRunner: indexerRunner,
		interval:      interval,
		runTimeout:    runTimeout,
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
		result, err := r.indexerRunner.RunIndexerOnce(runCtx)
		cancel()

		// Do not stop the runner on a single failure.
		// The next loop will retry from the current DB latest block.
		// Since DBLatest only advances after a successful insert,
		// failed blocks will not be skipped.
		if err != nil {
			switch {
			case errors.Is(err, types.ErrRequestCanceled):
				log.Printf("[indexer-runner] warn: canceled: %v\n", err)
			case errors.Is(err, types.ErrRPCTimeout), errors.Is(err, types.ErrDBTimeout):
				log.Printf("[indexer-runner] error: timeout: %v\n", err)
			default:
				log.Printf("[indexer-runner] error: %v\n", err)
			}
		} else {
			dbLatest := "null"
			if result.DBLatest != nil {
				dbLatest = fmt.Sprintf("%d", *result.DBLatest)
			}
			log.Printf(
				"[indexer-runner] sync_target=%s db=%s rpc_target=%d next=%d synced=%v\n",
				result.SyncTarget,
				dbLatest,
				result.RPCTarget,
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
