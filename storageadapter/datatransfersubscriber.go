package storageadapter

import (
	"fmt"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-markets/storagemarket/impl/requestvalidation"
)

// EventReceiver is any thing that can receive FSM events
type TransferProcess interface {
	//have many receiver function
	HandleCompleteFor(proposalid cid.Cid) error
	HandleCancelForDeal(proposalid cid.Cid) error
	HandleRestartForDeal(proposalid cid.Cid, channelId datatransfer.ChannelID) error
	HandleStalledForDeal(proposalid cid.Cid) error
	HandleInitForDeal(proposalid cid.Cid, channel datatransfer.ChannelID) error
	HandleFailedForDeal(proposalid cid.Cid, reason error) error
}

// ProviderDataTransferSubscriber is the function called when an event occurs in a data
// transfer received by a provider -- it reads the voucher to verify this event occurred
// in a storage market deal, then, based on the data transfer event that occurred, it generates
// and update message for the deal -- either moving to staged for a completion
// event or moving to error if a data transfer error occurs
func ProviderDataTransferSubscriber(deals TransferProcess) datatransfer.Subscriber {
	return func(event datatransfer.Event, channelState datatransfer.ChannelState) {
		voucher, ok := channelState.Voucher().(*requestvalidation.StorageDataTransferVoucher)
		// if this event is for a transfer not related to storage, ignore
		if !ok {
			log.Debugw("ignoring data-transfer event as it's not storage related", "event", datatransfer.Events[event.Code], "channelID",
				channelState.ChannelID())
			return
		}

		log.Debugw("processing storage provider dt event", "event", datatransfer.Events[event.Code], "proposalCid", voucher.Proposal, "channelID",
			channelState.ChannelID(), "channelState", datatransfer.Statuses[channelState.Status()])

		if channelState.Status() == datatransfer.Completed {
			//on complete
			err := deals.HandleCompleteFor(voucher.Proposal)
			if err != nil {
				log.Errorf("processing dt event: %s", err)
			}
		}

		// Translate from data transfer events to provider FSM events
		// Note: We ignore data transfer progress events (they do not affect deal state)
		err := func() error {
			switch event.Code {
			case datatransfer.Cancel:
				return deals.HandleCancelForDeal(voucher.Proposal)
			case datatransfer.Restart:
				return deals.HandleRestartForDeal(voucher.Proposal, channelState.ChannelID())
			case datatransfer.Disconnected:
				return deals.HandleStalledForDeal(voucher.Proposal)
			case datatransfer.Open:
				return deals.HandleInitForDeal(voucher.Proposal, channelState.ChannelID())
			case datatransfer.Error:
				return deals.HandleFailedForDeal(voucher.Proposal, fmt.Errorf("deal data transfer failed: %s", event.Message))
			default:
				return nil
			}
		}()
		if err != nil {
			log.Errorw("error processing storage provider dt event", "event", datatransfer.Events[event.Code], "proposalCid", voucher.Proposal, "channelID",
				channelState.ChannelID(), "err", err)
		}
	}
}
