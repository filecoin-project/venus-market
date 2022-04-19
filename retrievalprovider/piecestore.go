package retrievalprovider

import (
	"context"

	"github.com/filecoin-project/go-fil-markets/stores"
	"github.com/filecoin-project/venus-market/models/repo"
	"github.com/filecoin-project/venus-market/storageprovider"
	types "github.com/filecoin-project/venus/venus-shared/types/market"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

type PieceInfo struct {
	dagstore stores.DAGStoreWrapper
	dealRepo repo.StorageDealRepo
}

func (pinfo *PieceInfo) GetPieceInfoFromCid(ctx context.Context, payloadCID cid.Cid, piececid *cid.Cid) ([]*types.MinerDeal, error) {
	if piececid != nil && (*piececid).Defined() {
		minerDeals, err := pinfo.dealRepo.GetDealsByPieceCidAndStatus(ctx, (*piececid), storageprovider.ReadyRetrievalDealStatus...)
		if err != nil {
			return nil, err
		}
		return minerDeals, nil
	}

	// Get all pieces that contain the target block
	piecesWithTargetBlock, err := pinfo.dagstore.GetPiecesContainingBlock(payloadCID)
	if err != nil {
		return nil, xerrors.Errorf("getting pieces for cid %s: %w", payloadCID, err)
	}

	var allMinerDeals []*types.MinerDeal
	for _, pieceWithTargetBlock := range piecesWithTargetBlock {
		minerDeals, err := pinfo.dealRepo.GetDealsByPieceCidAndStatus(ctx, pieceWithTargetBlock, storageprovider.ReadyRetrievalDealStatus...)
		if err != nil {
			return nil, err
		}
		allMinerDeals = append(allMinerDeals, minerDeals...)
	}
	if len(allMinerDeals) > 0 {
		return allMinerDeals, nil
	}
	return nil, xerrors.Errorf("unable to find ready data for piece (%s) payload (%s)", piececid, payloadCID)
}
