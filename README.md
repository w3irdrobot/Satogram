# Satogram
Keysend to everyone on the lightning network. Works for LND nodes.

## What
Connects to your LND node, grabs your node's view of all nodes on the network, and keysends to each node. The keysend will include a custom message and amount in sats of your choosing (soonTM).

## How
Run the program using the flags specified in the below example. Specify the `host` to reach your node.  Ensure you point to the correct locations for the tls.cert and admin.macroon files for the `tls-path` and `macaroon-path` flags respectively. Specify the `network` to which your node will communicate over.

NOTE: the message field I haven't figured out yet, it will always send with some prebaked message (I forget what it is) until that is solved

Run with `go run cmd/main.go --host=localhost:10001 --tls-path=./tls.cert --macaroon-path=./admin.macaroon --network=regtest --message='gm lightning network!' --amount=1`