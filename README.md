# Satogram
Keysend to everyone on the lightning network. Works for LND nodes.

## What
Connects to your LND node, grabs your node's view of all nodes on the network, and keysends to each node. The keysend will include a custom message and amount in sats of your choosing.

## How
Run the program using the flags specified in the below example. Specify the `host` to reach your node.  Ensure you point to the correct locations for the tls.cert and admin.macroon files for the `tls-path` and `macaroon-path` flags respectively. Specify the `network` to which your node will communicate over.

Run with `go run cmd/main.go --host=localhost:10001 --tls-path=./tls.cert --macaroon-path=./admin.macaroon --network=regtest --message='gm lightning network!' --amount=1`


## Note
The message field I'm unclear if I'm using correctly, the format is quite confusing. If you use the message `gm lightning network!` the payment custom records will end up looknig like this from `lncli listpayments`:

```
"custom_records": {
  "34349334": "676d206c696768746e696e67206e6574776f726b21",
  "5482373484": "7062c58abe7ced6beb5929ae5be35b95dcfcd06ab91c380c8d4a1272e85f0943"
}
```
If you do a `hex.DecodeString("676d206c696768746e696e67206e6574776f726b21")` and then stringify the result of that you get back: `gm lightning network!`

Here is that bit in code: https://go.dev/play/p/IbX2QolRkbp

If this format is wrong please tell me and I will fix.