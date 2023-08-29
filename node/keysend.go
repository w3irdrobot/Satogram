package node

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
)

func (lnd *LND) Keysend(ctx context.Context, sats int64, message []byte) error {
	counterKickedOff := 0
	counterFinished := 0
	successes := 0
	failures := 0

	allStoreKeys := []string{}
	err := lnd.store.Keys("pk", func(key string) (bool, error) {
		allStoreKeys = append(allStoreKeys, key)
		return true, nil
	})

	numWorkers := len(allStoreKeys)
	fmt.Printf("number of nodes sending to: %d\n", numWorkers)
	time.Sleep(time.Second * 5)
	// lncli sendpayment --keysend --dest <pubkey> --amt 25000
	fn := func(pk string, client routerrpc.Router_SendPaymentV2Client) {
		for {
			fmt.Printf("Attempting keysend to: %s\n", pk)
			payment, err := client.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				fmt.Printf("error with send payment client for pk: %s with error: %s", pk, err.Error())
				return
			}

			// continue receiving updated while IN_FLIGHT
			switch payment.Status {
			case lnrpc.Payment_SUCCEEDED:
				successes++
				fmt.Printf("pubkey: %s status: %s total successed: %d\n", pk, payment.Status.String(), successes)
				return
			case lnrpc.Payment_IN_FLIGHT:
				fmt.Printf("pubkey: %s status: %s failure-reason: %s\n", pk, payment.Status.String(), payment.FailureReason.String())
			case lnrpc.Payment_FAILED:
				failures++
				fmt.Printf("pubkey: %s status: %s total failures: %d\n", pk, payment.Status.String(), failures)
				return
			default:
				fmt.Printf("pubkey: %s status: %s\n", pk, payment.Status.String())
			}
		}
	}

	keysend := func(boltKey string, wg *sync.WaitGroup) {
		defer wg.Done()
		counterKickedOff++
		fmt.Printf("on item #%d of %d. key: %s\n", counterKickedOff, numWorkers, boltKey)

		select {
		case <-ctx.Done():
			return
		default:
		}
		splitKey := strings.Split(boltKey, "pk ")
		if len(splitKey) != 2 {
			fmt.Printf("error parsing bolt key: %s", boltKey)
			return
		}
		pk := splitKey[1]
		randomBytes := make([]byte, 32)
		_, err := rand.Read(randomBytes)
		if err != nil {
			fmt.Println("Error generating random bytes:", err)
			return
		}

		preImage := hex.EncodeToString(randomBytes)

		hash := sha256.Sum256(randomBytes)
		paymentHash := hex.EncodeToString(hash[:])

		preImageFinal, err := hex.DecodeString(preImage)
		if err != nil {
			fmt.Printf("error doing preImageFinal: %s", err.Error())
			return
		}

		paymentHashFinal, err := hex.DecodeString(paymentHash)
		if err != nil {
			fmt.Printf("error doing paymenthashfinal: %s", err.Error())
			return
		}

		pkString, err := hex.DecodeString(pk)
		if err != nil {
			fmt.Printf("error decoing pk: %s with error: %s", pk, err.Error())
			return
		}
		client, err := lnd.router.SendPaymentV2(ctx, &routerrpc.SendPaymentRequest{
			Amt: sats,
			DestCustomRecords: map[uint64][]byte{
				5482373484: []byte(preImageFinal),
				34349334:   message,
			},
			TimeoutSeconds: 60,
			Dest:           pkString,
			FeeLimitSat:    20,
			PaymentHash:    []byte(paymentHashFinal),
		})
		if err != nil {
			fmt.Printf("error calling SendPaymentV2: %s", err.Error())
			return
		}
		fn(pk, client)
		counterFinished++
		fmt.Printf("finished counter: %d of total workers: %d\n", counterFinished, numWorkers)
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go keysend(allStoreKeys[i], &wg)
		time.Sleep(time.Millisecond * 50)
	}

	wg.Wait()

	if err != nil {
		return fmt.Errorf("error processing bolt keys for keysending: %s", err.Error())
	}
	return nil
}
