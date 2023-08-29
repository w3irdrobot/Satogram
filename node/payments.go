package node

import (
	"context"
	"fmt"

	"github.com/lightningnetwork/lnd/lnrpc"
)

func (lnd *LND) GetSuccessfulPaymentHistoryPubkeys(ctx context.Context) []string {
	pubkeysToExclude := []string{}
	// 0 will start from your first payment
	offset := uint64(0)
	for {
		payments, err := lnd.client.ListPayments(ctx, &lnrpc.ListPaymentsRequest{
			IncludeIncomplete: false,
			MaxPayments:       100,
			IndexOffset:       offset,
		})
		if err != nil {

			fmt.Printf("error calling lnrpc.Listpayments with offset: %d error: %s", offset, err.Error())
			return pubkeysToExclude
		}
		if len(payments.GetPayments()) == 0 {
			break
		}
		for _, payment := range payments.GetPayments() {
			for _, htlc := range payment.GetHtlcs() {
				if htlc.Status == lnrpc.HTLCAttempt_SUCCEEDED {
					lastHop := htlc.Route.GetHops()[len(htlc.Route.Hops)-1]
					if lastHop.CustomRecords[34349334] != nil {
						pubkeysToExclude = append(pubkeysToExclude, lastHop.PubKey)
					}
				}
			}
		}
		offset = payments.LastIndexOffset
	}

	return pubkeysToExclude
}
