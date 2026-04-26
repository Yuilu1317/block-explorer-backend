package service

import (
	"block-explorer-backend/internal/db/models"
	"block-explorer-backend/internal/types"
	"context"
	"errors"
	"math/big"
	"testing"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type fakeRPC struct {
	block *ethtypes.Block
	err   error

	blocks map[uint64]*ethtypes.Block
	errs   map[uint64]error

	onGetBlock func(number uint64)
}

func (f *fakeRPC) GetBlockByNumber(ctx context.Context, number uint64) (*ethtypes.Block, error) {
	if f.onGetBlock != nil {
		f.onGetBlock(number)
	}

	if f.errs != nil {
		if err, ok := f.errs[number]; ok {
			return nil, err
		}
	}
	if f.blocks != nil {
		if block, ok := f.blocks[number]; ok {
			return block, nil
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.block, f.err
}

type fakeBlockRepo struct {
	block *models.Block
	found bool
	err   error

	inserted       bool
	insertArg      *models.Block
	insertErr      error
	insertedBlocks []uint64
}

func (f *fakeBlockRepo) InsertBlock(ctx context.Context, block *models.Block) error {
	f.inserted = true
	f.insertArg = block

	if block != nil {
		f.insertedBlocks = append(f.insertedBlocks, block.Number)
	}

	return f.insertErr
}

func (f *fakeBlockRepo) GetBlockByNumber(ctx context.Context, number uint64) (*models.Block, bool, error) {
	return f.block, f.found, f.err
}

func setupTestService(t *testing.T) (*BlockService, *fakeBlockRepo, *fakeRPC) {
	t.Helper()

	rpc := &fakeRPC{}
	r := &fakeBlockRepo{}

	svc := NewBlockService(rpc, r)

	return svc, r, rpc
}

func TestBlockService_GetBlockByNumber_DBHit(t *testing.T) {
	svc, repo, _ := setupTestService(t)

	repo.block = &models.Block{
		Number: 20,
		Hash:   "0x123",
	}
	repo.found = true
	repo.err = nil

	result, err := svc.GetBlockByNumber(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Block.Number != 20 {
		t.Fatalf("expected block number 20, got %d", result.Block.Number)
	}
	if result.Block.Hash != "0x123" {
		t.Fatalf("expected block hash \"0x123\", got %s", result.Block.Hash)
	}
}

func TestBlockService_GetBlockByNumber_DBError(t *testing.T) {
	svc, repo, _ := setupTestService(t)

	repo.block = nil
	repo.found = false
	repo.err = types.ErrDBTimeout

	_, err := svc.GetBlockByNumber(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrDBTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestBlockService_GetBlockByNumber_RPCSuccess(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	repo.block = nil
	repo.found = false
	repo.err = nil

	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number: big.NewInt(20),
	})
	rpc.err = nil

	result, err := svc.GetBlockByNumber(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Block.Number != 20 {
		t.Fatalf("expected block number 20, got %d", result.Block.Number)
	}
}

func TestBlockService_GetBlockByNumber_RPCNotFound(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	repo.block = nil
	repo.found = false
	repo.err = nil

	rpc.err = types.ErrBlockNotFound
	_, err := svc.GetBlockByNumber(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrBlockNotFound) {
		t.Fatalf("expected RPCNotFound, got %v", err)
	}
}

func TestBlockService_SyncBlockToDB_Success(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number: big.NewInt(20),
	})
	rpc.err = nil
	err := svc.SyncBlockToDB(context.Background(), 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !repo.inserted {
		t.Fatalf("expected InsertBlock to be called")
	}

	if repo.insertArg == nil {
		t.Fatalf("expected inserted block, got nil")
	}

	if repo.insertArg.Number != 20 {
		t.Fatalf("expected inserted block number=20, got %d", repo.insertArg.Number)
	}
}

func TestBlockService_SyncBlockToDB_RPCNotFound(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpc.block = nil
	rpc.err = types.ErrBlockNotFound
	err := svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrBlockNotFound) {
		t.Fatalf("expected ErrBlockNotFound, got %v", err)
	}
	if repo.inserted {
		t.Fatalf("expected InsertBlock NOT to be called")
	}
}

func TestBlockService_SyncBlockToDB_RPCError(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpc.block = nil
	rpc.err = types.ErrRPCTimeout
	err := svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrRPCTimeout) {
		t.Fatalf("expected ErrRPCTimeout, got %v", err)
	}
	if repo.inserted {
		t.Fatalf("expected InsertBlock NOT to be called")
	}
}

func TestBlockService_SyncBlockToDB_InsertError(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpc.block = ethtypes.NewBlockWithHeader(&ethtypes.Header{
		Number: big.NewInt(20),
	})
	rpc.err = nil

	repo.insertErr = errors.New("some error")

	err := svc.SyncBlockToDB(context.Background(), 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !repo.inserted {
		t.Fatalf("expected InsertBlock to be called")
	}
	if repo.insertArg == nil {
		t.Fatalf("expected insertArg not nil")
	}
	if repo.insertArg.Number != 20 {
		t.Fatalf("expected inserted block number=20, got %d", repo.insertArg.Number)
	}
}

func TestBlockService_SyncBlockRangeToDB_AllSuccess(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpc.blocks = map[uint64]*ethtypes.Block{
		10: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(10)}),
		11: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(11)}),
		12: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(12)}),
	}

	result, err := svc.SyncBlockRangeToDB(context.Background(), 10, 12)
	switch {
	case err != nil:
		t.Fatalf("expected nil error, got %v", err)
	case result.Requested != 3:
		t.Fatalf("expected requested=3, got %d", result.Requested)
	case result.Succeeded != 3:
		t.Fatalf("expected succeeded=3, got %d", result.Succeeded)
	case result.Failed != 0:
		t.Fatalf("expected failed=0, got %d", result.Failed)
	case len(repo.insertedBlocks) != 3:
		t.Fatalf("expected 3 inserted blocks, got %d", len(repo.insertedBlocks))
	}
}

func TestBlockService_SyncBlockRangeToDB_ErrInvalidBlockRange(t *testing.T) {
	svc, _, _ := setupTestService(t)

	result, err := svc.SyncBlockRangeToDB(context.Background(), 10, 9)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrInvalidBlockRange) {
		t.Fatalf("expected ErrInvalidBlockRange, got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil block, got %v", result)
	}
}

func TestBlockService_SyncBlockRangeToDB_ErrBlockRangeTooLarge(t *testing.T) {
	svc, _, _ := setupTestService(t)

	result, err := svc.SyncBlockRangeToDB(context.Background(), 1, 200)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrBlockRangeTooLarge) {
		t.Fatalf("expected ErrBlockRangeTooLarge, got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil block, got %v", result)
	}
}

func TestBlockService_SyncBlockRangeToDB_Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, repo, rpc := setupTestService(t)

	rpc.blocks = map[uint64]*ethtypes.Block{
		10: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(10)}),
		11: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(11)}),
		12: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(12)}),
	}

	rpc.onGetBlock = func(number uint64) {
		if number == 11 {
			cancel()
		}
	}

	result, err := svc.SyncBlockRangeToDB(ctx, 10, 12)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, types.ErrRequestCanceled) {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
	if result.Succeeded != 2 {
		t.Fatalf("expected succeeded=2, got %d", result.Succeeded)
	}
	if len(repo.insertedBlocks) != 2 {
		t.Fatalf("expected 2 inserted blocks, got %d", len(repo.insertedBlocks))
	}
}
func TestBlockService_SyncBlockRangeToDB_PartialFailure(t *testing.T) {
	svc, repo, rpc := setupTestService(t)

	rpc.blocks = map[uint64]*ethtypes.Block{
		10: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(10)}),
		12: ethtypes.NewBlockWithHeader(&ethtypes.Header{Number: big.NewInt(12)}),
	}
	rpc.errs = map[uint64]error{
		11: errors.New("some error"),
	}
	result, err := svc.SyncBlockRangeToDB(context.Background(), 10, 12)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Succeeded != 2 {
		t.Fatalf("expected succeeded=2, got %d", result.Succeeded)
	}
	if result.Failed != 1 {
		t.Fatalf("expected failed=1, got %d", result.Failed)
	}
	if len(result.FailedBlocks) != 1 || result.FailedBlocks[0] != 11 {
		t.Fatalf("expected failed block 11, got %+v", result.FailedBlocks)
	}
	if len(repo.insertedBlocks) != 2 {
		t.Fatalf("expected 2 inserted blocks, got %d", len(repo.insertedBlocks))
	}
}
