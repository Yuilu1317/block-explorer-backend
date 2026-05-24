package indexer

import (
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"testing"
)

type fakeBlockRPC struct {
	targetNumber uint64
	err          error

	getTargetCalled bool
	gotTag          string
}

func (f *fakeBlockRPC) GetBlockNumberByTag(ctx context.Context, tag string) (uint64, error) {
	f.getTargetCalled = true
	f.gotTag = tag
	return f.targetNumber, f.err
}

type fakeBlockRepository struct {
	latestNumber uint64
	exists       bool
	err          error

	getLatestCalled bool
}

func (f *fakeBlockRepository) GetLatestFullySyncedBlockNumber(ctx context.Context) (uint64, bool, error) {
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
	return setupTestIndexerWithConfig(t, "safe", 0)
}

func setupTestIndexerWithConfig(t *testing.T, syncTarget string, startBlock uint64) (*BlockIndexer, *fakeBlockRPC, *fakeBlockRepository, *fakeBlockService) {
	t.Helper()

	rpc := &fakeBlockRPC{}
	repo := &fakeBlockRepository{}
	service := &fakeBlockService{}

	svc := NewBlockIndexer(rpc, repo, service, syncTarget, startBlock)

	return svc, rpc, repo, service
}

func TestBlockIndexer_GetNextBlockToSync_ReturnsDBLatestPlusOne(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexer(t)
	ctx := context.Background()
	rpc.targetNumber = 1
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
	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("GetNextBlockToSync(): expected RPCTarget=1, got %v", result.RPCTarget)
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
	rpc.targetNumber = 2
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
	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("GetNextBlockToSync(): expected RPCTarget=2, got %v", result.RPCTarget)
	}
	if result.Next != repo.latestNumber+1 {
		t.Fatalf("GetNextBlockToSync(): expected Next=2, got %v", result.Next)
	}
	if !result.ShouldSync {
		t.Fatalf("GetNextBlockToSync(): expected ShouldSync=true, got false")
	}
}

func TestBlockIndexer_GetNextBlockToSync_StartsFromConfiguredStartBlockWhenDBIsEmpty(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexerWithConfig(t, "safe", 100)

	ctx := context.Background()
	rpc.targetNumber = 150
	repo.exists = false

	result, err := svc.GetNextBlockToSync(ctx)
	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}

	if result.DBLatest != nil {
		t.Fatalf("GetNextBlockToSync(): expected DBLatest nil, got %d", *result.DBLatest)
	}

	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("GetNextBlockToSync(): expected RPCTarget=%d, got %d", rpc.targetNumber, result.RPCTarget)
	}

	if result.Next != 100 {
		t.Fatalf("GetNextBlockToSync(): expected Next=100, got %d", result.Next)
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
	if rpc.getTargetCalled {
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
	if !rpc.getTargetCalled {
		t.Fatalf("expected RPC to be called")
	}
}

func TestBlockIndexer_RunIndexerOnce_SyncsNextBlockWhenShouldSyncTrue(t *testing.T) {
	svc, rpc, repo, service := setupTestIndexer(t)
	ctx := context.Background()
	repo.latestNumber = 1
	repo.exists = true
	rpc.targetNumber = 2

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
	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("RunIndexerOnce(): expected RPCTarget=%d, got %d", rpc.targetNumber, result.RPCTarget)
	}
	if result.SyncTarget != "safe" {
		t.Fatalf("RunIndexerOnce(): expected SyncTarget=safe, got %s", result.SyncTarget)
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
	rpc.targetNumber = 1
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
	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("RunIndexerOnce(): expected RPCTarget=%d, got %d", rpc.targetNumber, result.RPCTarget)
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
	if !rpc.getTargetCalled {
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
	rpc.targetNumber = 2
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
	if !rpc.getTargetCalled {
		t.Fatalf("expected RPC to be called before sync")
	}
	if !service.syncCalled {
		t.Fatalf("expected Sync=true, got false")
	}
}

func TestBlockIndexer_GetNextBlockToSync_UsesSyncTarget(t *testing.T) {
	svc, rpc, repo, _ := setupTestIndexerWithConfig(t, "finalized", 0)

	ctx := context.Background()
	repo.latestNumber = 10
	repo.exists = true
	rpc.targetNumber = 20

	result, err := svc.GetNextBlockToSync(ctx)
	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}

	if !rpc.getTargetCalled {
		t.Fatalf("expected RPC target to be called")
	}

	if rpc.gotTag != "finalized" {
		t.Fatalf("expected sync target finalized, got %s", rpc.gotTag)
	}

	if result.SyncTarget != "finalized" {
		t.Fatalf("expected result SyncTarget finalized, got %s", result.SyncTarget)
	}

	if result.RPCTarget != rpc.targetNumber {
		t.Fatalf("expected RPCTarget=%d, got %d", rpc.targetNumber, result.RPCTarget)
	}
}

func TestBlockIndexer_GetNextBlockToSync_UsesLatestFullySyncedBlockAsCursor(t *testing.T) {
	indexer, rpc, repo, _ := setupTestIndexer(t)
	ctx := context.Background()

	repo.latestNumber = 99
	repo.exists = true
	rpc.targetNumber = 105

	result, err := indexer.GetNextBlockToSync(ctx)
	if err != nil {
		t.Fatalf("GetNextBlockToSync(): %v", err)
	}

	if !repo.getLatestCalled {
		t.Fatalf("expected GetLatestFullySyncedBlockNumber to be called")
	}

	if !rpc.getTargetCalled {
		t.Fatalf("expected RPC target to be called")
	}

	if result.DBLatest == nil {
		t.Fatalf("expected DBLatest=%d, got nil", repo.latestNumber)
	}

	if *result.DBLatest != 99 {
		t.Fatalf("expected DBLatest=99, got %d", *result.DBLatest)
	}

	if result.RPCTarget != 105 {
		t.Fatalf("expected RPCTarget=105, got %d", result.RPCTarget)
	}

	if result.Next != 100 {
		t.Fatalf("expected Next=100, got %d", result.Next)
	}

	if !result.ShouldSync {
		t.Fatalf("expected ShouldSync=true")
	}
}
