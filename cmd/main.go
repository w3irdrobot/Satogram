package main

import (
	"Satogram/node"
	"Satogram/storage"
	"context"
	"flag"
	"fmt"
	"path"

	"github.com/go-errors/errors"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	var host, tlsPath, macaroonPath, network, message, excludePath string
	var amount int64

	flag.StringVar(&host, "host", "localhost:10001", "Specify node's host address")
	flag.StringVar(&tlsPath, "tls-path", "./tls.cert", "Specify absolute path to tls.cert")
	flag.StringVar(&macaroonPath, "macaroon-path", "./admin.macaroon", "Specify absolute path to admin.macaroon")
	flag.StringVar(&network, "network", "regtest", "Specify network node is to use (e.g. mainnet, testnet, regtest)")
	flag.StringVar(&message, "message", "gm lightning network!", "Specify message to send in the keysend payment")
	flag.Int64Var(&amount, "amount", 1, "Specify amount in sats to spend for each keysend")
	flag.StringVar(&excludePath, "exclude-pubkeys-path", "exclude-pubkeys.txt", "Specify path to .txt file of pubkeys to exclude keysending to")
	flag.Parse()
	// setup the storage
	store, err := storage.NewBolt(path.Join("./", "satogram.db"))
	if err != nil {
		fmt.Printf("error creating bolt storage: %s\n", err.Error())
		return
	}

	// loading in bolt keys
	store.Keys("pk ", func(key string) (bool, error) {
		alias, err := store.Get(key)
		if err != nil {
			return false, fmt.Errorf("error getting value from store with key: %s and error: %w", key, err)
		}
		fmt.Printf("got stored key: %s  alias: %s\n", key, string(alias))
		return true, nil
	})

	lnd, err := node.NewLND(store, host, tlsPath, macaroonPath, network, excludePath)
	if err != nil {
		fmt.Printf("Error creating lnd struct: %s\n", err.Error())
		return
	}

	err = lnd.EstablishClientConnections(ctx)
	if err != nil {
		fmt.Printf("error establishing client connection: %s\n", err.Error())
		return
	}
	err = lnd.Ping(ctx)
	if err != nil {
		fmt.Printf("error pinging node: %s\n", err.Error())
		return
	}
	pubkeys, err := lnd.GetNodes(ctx)
	if err != nil {
		fmt.Printf("failed to get pubkeys: %s\n", err.Error())
		return
	}

	newKeysAdded := 0
	for pk, alias := range pubkeys {
		val, err := store.Get(fmt.Sprintf("pk %s", pk))
		if err != nil && !errors.Is(storage.ErrNotFound, err) {
			fmt.Printf("error checking on pubkey from store (%s) error: %s\n", pk, err.Error())
			continue
		}
		if val == nil {
			fmt.Printf("adding pubkey: %s alias: %s\n", pk, alias)
			err = store.Set(fmt.Sprintf("pk %s", pk), []byte(alias))
			if err != nil {
				fmt.Printf("error adding pk (%s) to store: %s\n", pk, err.Error())
				continue
			}
			newKeysAdded++
		}
	}
	numNodes, err := store.NumItems()
	if err != nil {
		fmt.Printf("error getting number of nodes: %s\n", err.Error())
		return
	}
	fmt.Printf("%d new pubkeys added, %d stored in the db\n", newKeysAdded, numNodes)
	err = lnd.Keysend(ctx, amount, []byte(message))
	if err != nil {
		fmt.Printf("error executing keysend: %s\n", err.Error())
	}
	cancel()
}
