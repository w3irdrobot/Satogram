package node

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
)

func (lnd *LND) Keysend(ctx context.Context, sats int64, message string) error {
	// lncli sendpayment --keysend --dest <pubkey> --amt 25000
	fn := func(pk string, client routerrpc.Router_SendPaymentV2Client) error {
		for {
			payment, err := client.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("error with send payment client: %s", err.Error())
			}

			// continue receiving updated while IN_FLIGHT
			if payment.Status != lnrpc.Payment_IN_FLIGHT {
				return nil
			}
		}
	}

	keysend := func(boltKey string) (bool, error) {
		select {
		case <-ctx.Done():
			return true, nil
		default:
		}
		splitKey := strings.Split(boltKey, "pk ")
		if len(splitKey) != 2 {
			return false, fmt.Errorf("error parsing bolt key: %s", boltKey)
		}
		pk := splitKey[1]
		randomBytes := make([]byte, 32)
		_, err := rand.Read(randomBytes)
		if err != nil {
			fmt.Println("Error generating random bytes:", err)
			return false, err
		}

		preImage := hex.EncodeToString(randomBytes)

		hash := sha256.Sum256(randomBytes)
		paymentHash := hex.EncodeToString(hash[:])

		preImageFinal, err := hex.DecodeString(preImage)
		if err != nil {
			return false, fmt.Errorf("error doing preImageFinal: %s", err.Error())
		}

		paymentHashFinal, err := hex.DecodeString(paymentHash)
		if err != nil {
			return false, fmt.Errorf("error doing paymenthashfinal: %s", err.Error())
		}

		pkString, err := hex.DecodeString(pk)
		if err != nil {
			return false, fmt.Errorf("error decoing pk: %s with error: %s", pk, err.Error())
		}

		client, err := lnd.router.SendPaymentV2(ctx, &routerrpc.SendPaymentRequest{
			Amt: sats,
			DestCustomRecords: map[uint64][]byte{
				5482373484: []byte(preImageFinal),
				34349334:   []byte(message),
			},
			TimeoutSeconds: 10,
			Dest:           pkString,
			FeeLimitSat:    10,
			PaymentHash:    []byte(paymentHashFinal),
		})
		if err != nil {

			return false, err
		}
		if err := fn(pk, client); err != nil {
			return false, err
		}

		return true, nil
	}
	lnd.store.Keys("pk", keysend)

	return nil
}
