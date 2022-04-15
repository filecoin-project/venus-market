package dagstore

import (
	"context"
	"fmt"
	"io"

	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/dagstore/throttle"
	"github.com/filecoin-project/go-padreader"

	"github.com/filecoin-project/venus-market/models/repo"
	"github.com/filecoin-project/venus-market/piecestorage"
)

type MarketAPI interface {
	FetchUnsealedPiece(ctx context.Context, pieceCid cid.Cid) (io.ReadCloser, error)
	GetUnpaddedCARSize(ctx context.Context, pieceCid cid.Cid) (uint64, error)
	IsUnsealed(ctx context.Context, pieceCid cid.Cid) (bool, error)
	Start(ctx context.Context) error
}

type marketAPI struct {
	pieceStorageMgr *piecestorage.PieceStorageManager
	pieceRepo       repo.StorageDealRepo
	throttle        throttle.Throttler
}

var _ MarketAPI = (*marketAPI)(nil)

func NewMinerAPI(repo repo.Repo, pieceStorageMgr *piecestorage.PieceStorageManager, concurrency int) MarketAPI {
	return &marketAPI{
		pieceRepo:       repo.StorageDealRepo(),
		pieceStorageMgr: pieceStorageMgr,
		throttle:        throttle.Fixed(concurrency),
	}
}

func (m *marketAPI) Start(_ context.Context) error {
	return nil
}

func (m *marketAPI) IsUnsealed(ctx context.Context, pieceCid cid.Cid) (bool, error) {
	_, err := m.pieceStorageMgr.SelectStorageForRead(ctx, pieceCid.String())
	if err != nil {
		return false, fmt.Errorf("unable to find storage for piece %s %w", pieceCid, err)
	}
	return true, nil
	//todo check isunseal from miner
}

func (m *marketAPI) FetchUnsealedPiece(ctx context.Context, pieceCid cid.Cid) (io.ReadCloser, error) {
	payloadSize, pieceSize, err := m.pieceRepo.GetPieceSize(ctx, pieceCid)
	if err != nil {
		return nil, err
	}

	pieceStorage, err := m.pieceStorageMgr.SelectStorageForRead(ctx, pieceCid.String())
	if err != nil {
		// todo unseal: ask miner who have this data, send unseal cmd, and read and pay after receive data
		// 1. select miner
		// 2. send unseal request
		// 3. receive data and return
		return nil, xerrors.Errorf("ask for child miner for piece data not impl")
	}
	r, err := pieceStorage.Read(ctx, pieceCid.String())
	if err != nil {
		return nil, err
	}
	padR, err := padreader.NewInflator(r, payloadSize, pieceSize.Unpadded())
	if err != nil {
		return nil, err
	}
	return iocloser{r, padR}, nil
}

func (m *marketAPI) GetUnpaddedCARSize(ctx context.Context, pieceCid cid.Cid) (uint64, error) {
	pieceInfo, err := m.pieceRepo.GetPieceInfo(ctx, pieceCid)
	if err != nil {
		return 0, xerrors.Errorf("failed to fetch pieceInfo for piece %s: %w", pieceCid, err)
	}

	if len(pieceInfo.Deals) == 0 {
		return 0, xerrors.Errorf("no storage deals found for piece %s", pieceCid)
	}

	len := pieceInfo.Deals[0].Length

	return uint64(len), nil
}

/*

func (p *pieceProvider) unsealPiece(ctx context.Context, dealInfo *piece.DealInfo, sector storage.SectorRef, offset types2.UnpaddedByteIndex, size abi.UnpaddedPieceSize) (io.ReadCloser, error) {
	pieceCid := dealInfo.Proposal.PieceCID
	pieceOffset := abi.UnpaddedPieceSize(offset) - dealInfo.Offset.Unpadded()
	if err := p.miner.SectorsUnsealPiece(ctx, address.Address(p.maddr), pieceCid, sector, offset.Padded(), size.Padded(), path.Join(string(*p.pieceStrorageCfg), pieceCid.String())); err != nil {
		log.Errorf("failed to SectorsUnsealPiece: %s", err)
		return nil, xerrors.Errorf("unsealing piece: %w", err)
	}

	//todo config
	ctx, _ = context.WithTimeout(ctx, time.Hour*6)
	tm := time.NewTimer(time.Second * 30)

	for {
		select {
		case <-tm.C:
			has, err := p.pieceStorage.Has(pieceCid.String())
			if err != nil {
				return nil, xerrors.Errorf("unable to check piece in piece stroage %w", err)
			}
			if has {
				goto LOOP
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
LOOP:
	//todo how to store data piece not completed piece
	log.Debugf("unsealed a sector file to read the piece, sector=%+v, offset=%d, size=%d", sector, offset, size)
	// move piece to storage
	r, err := p.pieceStorage.ReadOffset(ctx, pieceCid.String(), pieceOffset, size)
	if err != nil {
		log.Errorf("unable to read piece in piece storage;sector=%+v, piececid=%s err:%s", sector.ID, pieceCid, err)
		return nil, err
	}
	return r, err
}

*/

var _ io.ReadCloser = (*iocloser)(nil)

type iocloser struct {
	closeR io.ReadCloser
	readR  io.Reader
}

func (i iocloser) Read(p []byte) (n int, err error) {
	return i.readR.Read(p)
}

func (i iocloser) Close() error {
	return i.closeR.Close()
}
