package indexer

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"testing"
)

type fakeBlockRPC struct {
	latestNumber uint64
	err          error

	getLatestCalled bool
}

func (f *fakeBlockRPC) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	f.getLatestCalled = true
	return f.latestNumber, f.err
}

type fakeBlockRepository struct {
	latestNumber uint64
	exists       bool
	err          error

	getLatestCalled bool
}

func (f *fakeBlockRepository) GetLatestBlockNumber(ctx context.Context) (uint64, bool, error) {
	f.getLatestCalled = true
	return f.latestNumber, f.exists, f.err
}

type fakeBlockService struct {
	syncCalled  bool
	syncedBlock uint64
	err         error
}

func (f *fakeBlockService) SyncBlockToDB(ctx context.Context, number uint64) error {
	f.syncCalled = true
	f.syncedBlock = number
	return f.err
}

func setupTestIndexer(t *testing.T) (*BlockIndexer, *fakeBlockRPC, *fakeBlockRepository, *fakeBlockService) {
	t.Helper()

	rpc := &fakeBlockRPC{}
	repo := &fakeBlockRepository{}
	service := &fakeBlockService{}

	svc := NewBlockIndexer(rpc, repo, service)

	return svc, rpc, repo, service
}

func TestBlockIndexer_GetNextBlockToSync_ReturnsLatestBlockNumberPlusOne(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexer(t)
	ctx := context.Background()
	rpc.latestNumber = 1
	repo.latestNumber = 1
	repo.exists = true

	result, err := svc.GetNextBlockToSync(ctx)

	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}
	if result.DBLatest == nil {
		t.Fatalf("GetNextBlockToSync(): expected DBLatest=%d, got nil", repo.latestNumber)
	}
	if *result.DBLatest != repo.latestNumber {
		t.Fatalf("GetNextBlockToSync(): expected DBLatest=1, got %v", *result.DBLatest)
	}
	if result.RPCLatest != rpc.latestNumber {
		t.Fatalf("GetNextBlockToSync(): expected RPCLatest=1, got %v", result.RPCLatest)
	}
	if result.Next != repo.latestNumber+1 {
		t.Fatalf("GetNextBlockToSync(): expected Next=2, got %v", result.Next)
	}
	if result.ShouldSync {
		t.Fatalf("GetNextBlockToSync(): expected ShouldSync=false, got true")
	}
}

func TestBlockIndexer_GetNextBlockToSync_ShouldSyncWhenRPCIsAhead(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexer(t)
	ctx := context.Background()
	rpc.latestNumber = 2
	repo.latestNumber = 1
	repo.exists = true
	result, err := svc.GetNextBlockToSync(ctx)
	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}
	if result.DBLatest == nil {
		t.Fatalf("GetNextBlockToSync(): expected DBLatest=%d, got nil", repo.latestNumber)
	}
	if *result.DBLatest != repo.latestNumber {
		t.Fatalf("GetNextBlockToSync(): expected DBLatest=1, got %v", *result.DBLatest)
	}
	if result.RPCLatest != rpc.latestNumber {
		t.Fatalf("GetNextBlockToSync(): expected RPCLatest=2, got %v", result.RPCLatest)
	}
	if result.Next != repo.latestNumber+1 {
		t.Fatalf("GetNextBlockToSync(): expected Next=2, got %v", result.Next)
	}
	if !result.ShouldSync {
		t.Fatalf("GetNextBlockToSync(): expected ShouldSync=true, got false")
	}
}

func TestBlockIndexer_GetNextBlockToSync_StartsFromZeroWhenDBIsEmpty(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexer(t)
	ctx := context.Background()
	rpc.latestNumber = 5
	repo.exists = false
	result, err := svc.GetNextBlockToSync(ctx)
	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}
	if result.DBLatest != nil {
		t.Fatalf("GetNextBlockToSync(): expected DBLatest nil，got %d", *result.DBLatest)
	}
	if result.RPCLatest != rpc.latestNumber {
		t.Fatalf("GetNextBlockToSync(): expected RPCLatest=%d, got %d", rpc.latestNumber, result.RPCLatest)
	}
	if result.Next != 0 {
		t.Fatalf("GetNextBlockToSync(): expected Next=0, got %v", result.Next)
	}
	if !result.ShouldSync {
		t.Fatalf("GetNextBlockToSync(): expected ShouldSync=true, got false")
	}
}

func TestBlockIndexer_GetNextBlockToSync_ReturnsErrorWhenRepoFails(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexer(t)
	ctx := context.Background()
	repo.err = types.ErrDBTimeout
	result, err := svc.GetNextBlockToSync(ctx)
	if err == nil {
		t.Fatalf("GetNextBlockToSync(): expected error")
	}
	if result != nil {
		t.Fatalf("GetNextBlockToSync(): expected nil, got %v", result)
	}
	if !errors.Is(err, types.ErrDBTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if rpc.getLatestCalled {
		t.Fatalf("expected rpc not to be called when repo fails")
	}
}

func TestBlockIndexer_GetNextBlockToSync_ReturnsErrorWhenRPCFails(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexer(t)
	ctx := context.Background()
	repo.latestNumber = 1
	repo.exists = true
	rpc.err = types.ErrRPCTimeout
	result, err := svc.GetNextBlockToSync(ctx)
	if err == nil {
		t.Fatalf("GetNextBlockToSync(): expected error")
	}
	if result != nil {
		t.Fatalf("GetNextBlockToSync(): expected nil, got %v", result)
	}
	if !errors.Is(err, types.ErrRPCTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if !repo.getLatestCalled {
		t.Fatalf("expected DB to be called when rpc fails")
	}
	if !rpc.getLatestCalled {
		t.Fatalf("expected RPC to be called")
	}
}

func TestBlockIndexer_RunIndexerOnce_SyncsNextBlockWhenShouldSyncTrue(t *testing.T) {
	svc, rpc, repo, service := setupTestIndexer(t)
	ctx := context.Background()
	repo.latestNumber = 1
	repo.exists = true
	rpc.latestNumber = 2

	result, err := svc.RunIndexerOnce(ctx)
	if err != nil {
		t.Fatalf("RunIndexerOnce(): %v", err)
	}
	if result.DBLatest == nil {
		t.Fatalf("RunIndexerOnce(): expected DBLatest=%d, got nil", repo.latestNumber)
	}
	if *result.DBLatest != repo.latestNumber {
		t.Fatalf("RunIndexerOnce(): expected DBLatest=%d, got %d", repo.latestNumber, *result.DBLatest)
	}
	if result.RPCLatest != rpc.latestNumber {
		t.Fatalf("RunIndexerOnce(): expected RPCLatest=%d, got %d", rpc.latestNumber, result.RPCLatest)
	}
	wantNext := repo.latestNumber + 1
	if result.NextToSync != wantNext {
		t.Fatalf("RunIndexerOnce(): expected NextToSync=%d, got %d", wantNext, result.NextToSync)
	}
	if !service.syncCalled {
		t.Fatalf("RunIndexerOnce(): expected Sync=true, got false")
	}
	if service.syncedBlock != wantNext {
		t.Fatalf("RunIndexerOnce(): expected syncedBlock=%d, got %d", wantNext, service.syncedBlock)
	}
}

func TestBlockIndexer_RunIndexerOnce_DoesNotSyncWhenNoNewBlock(t *testing.T) {
	svc, rpc, repo, service := setupTestIndexer(t)
	ctx := context.Background()
	repo.latestNumber = 1
	repo.exists = true
	rpc.latestNumber = 1
	result, err := svc.RunIndexerOnce(ctx)
	if err != nil {
		t.Fatalf("RunIndexerOnce(): %v", err)
	}
	if result.DBLatest == nil {
		t.Fatalf("RunIndexerOnce(): expected DBLatest=%d, got nil", repo.latestNumber)
	}
	if *result.DBLatest != repo.latestNumber {
		t.Fatalf("RunIndexerOnce(): expected DBLatest=%d, got %d", repo.latestNumber, *result.DBLatest)
	}
	if result.RPCLatest != rpc.latestNumber {
		t.Fatalf("RunIndexerOnce(): expected RPCLatest=%d, got %d", rpc.latestNumber, result.RPCLatest)
	}
	wantNext := repo.latestNumber + 1
	if result.NextToSync != wantNext {
		t.Fatalf("RunIndexerOnce(): expected NextToSync=%d, got %d", wantNext, result.NextToSync)
	}
	if service.syncCalled {
		t.Fatalf("RunIndexerOnce(): expected Sync=false, got true")
	}
}

func TestBlockIndexer_RunIndexerOnce_ReturnsErrorWhenGetNextBlockFails(t *testing.T) {
	svc, rpc, repo, service := setupTestIndexer(t)
	ctx := context.Background()
	repo.latestNumber = 1
	repo.exists = true
	rpc.err = types.ErrRPCTimeout
	result, err := svc.RunIndexerOnce(ctx)
	if err == nil {
		t.Fatalf("RunIndexerOnce(): expected error")
	}
	if result != nil {
		t.Fatalf("RunIndexerOnce(): expected nil, got %v", result)
	}
	if !errors.Is(err, types.ErrRPCTimeout) {
		t.Fatalf("RunIndexerOnce(): expected timeout error, got %v", err)
	}
	if !rpc.getLatestCalled {
		t.Fatalf("expected RPC to be called after DB succeeds")
	}
	if service.syncCalled {
		t.Fatalf("expected Sync=false, got true")
	}
}

func TestBlockIndexer_RunIndexerOnce_ReturnsErrorWhenSyncFails(t *testing.T) {
	svc, rpc, repo, service := setupTestIndexer(t)
	ctx := context.Background()
	repo.latestNumber = 1
	repo.exists = true
	rpc.latestNumber = 2
	service.err = types.ErrDBTimeout
	result, err := svc.RunIndexerOnce(ctx)
	if err == nil {
		t.Fatalf("RunIndexerOnce(): expected error")
	}
	if result != nil {
		t.Fatalf("RunIndexerOnce(): expected nil, got %v", result)
	}
	if !errors.Is(err, types.ErrDBTimeout) {
		t.Fatalf("RunIndexerOnce(): expected timeout error, got %v", err)
	}
	wantNext := repo.latestNumber + 1
	if service.syncedBlock != wantNext {
		t.Fatalf("expected syncedBlock=%d, got %d", wantNext, service.syncedBlock)
	}
	if !rpc.getLatestCalled {
		t.Fatalf("expected RPC to be called before sync")
	}
	if !service.syncCalled {
		t.Fatalf("expected Sync=true, got false")
	}
}
