/*
## Protocols

The following RPC protocols are supported:

* URI over HTTP
* JSONRPC over HTTP
* JSONRPC over websockets

RPC is built using Tendermint's RPC library which contains its own set of documentation and tests.
See it here: https://github.com/tendermint/tendermint/tree/master/rpc/lib

## Configuration

RPC can be configured by tuning parameters under `[rpc]` table in the `$TMHOME/config/config.toml` file or by using the `--rpc.X` command-line flags.

Default rpc listen address is `tcp://0.0.0.0:27147`. To set another address,  set the `laddr` config parameter to desired value.
CORS (Cross-Origin Resource Sharing) can be enabled by setting `cors_allowed_origins`, `cors_allowed_methods`, `cors_allowed_headers` config parameters.

## Arguments

Arguments which expect strings or byte arrays may be passed as quoted strings, like `"abc"` or as `0x`-prefixed strings, like `0x616263`.

## URI/HTTP

```bash
curl 'localhost:27147/broadcast_tx_sync?tx="abc"'
```

> Response:

```json
{
	"error": "",
	"result": {
		"hash": "2B8EC32BA2579B3B8606E42C06DE2F7AFA2556EF",
		"log": "",
		"data": "",
		"code": "0"
	},
	"id": "",
	"jsonrpc": "2.0"
}
```

## JSONRPC/HTTP

JSONRPC requests can be POST'd to the root RPC endpoint via HTTP (e.g. http://localhost:27147/).

```json
{
	"method": "broadcast_tx_sync",
	"jsonrpc": "2.0",
	"params": [ "abc" ],
	"id": "dontcare"
}
```

## JSONRPC/websockets

JSONRPC requests can be made via websocket. The websocket endpoint is at `/websocket`, e.g. `localhost:27147/websocket`.  Asynchronous RPC functions like event `subscribe` and `unsubscribe` are only available via websockets.


## Get the list

An HTTP Get request to the root RPC endpoint shows a list of available endpoints.

```bash
curl 'localhost:27147'
```

> Response:

```plain
Available endpoints:
/abci_info
/dump_consensus_state
/genesis
/net_info
/num_unconfirmed_txs
/status
/health
/unconfirmed_txs
/unsafe_flush_mempool
/unsafe_stop_cpu_profiler
/validators

Endpoints that require arguments:
/abci_query?path=_&data=_&prove=_
/block?height=_
/blockchain?minHeight=_&maxHeight=_
/broadcast_tx_async?tx=_
/broadcast_tx_commit?tx=_
/broadcast_tx_sync?tx=_
/commit?height=_
/dial_seeds?seeds=_
/dial_persistent_peers?persistent_peers=_
/subscribe?event=_
/tx?hash=_&prove=_
/unsafe_start_cpu_profiler?filename=_
/unsafe_write_heap_profile?filename=_
/unsubscribe?event=_
```

# Endpoints
*/
package core
