# Node RPC

RPC endpoints may be used to interact with a node directly over HTTP or websockets. Using RPC, you may perform low-level operations like executing ABCI queries, viewing network/consensus state or broadcasting a transaction.

## Connecting

There are two main ways to connect to a node to send RPC commands.

### Use your own local node

This page assumes that you have your own node running locally, so examples here use `localhost:27146` to represent using RPC commands on a local node.

Alternatively, you are able to use a node that is hosted in the Binance Chain network.

### Use an existing node on the network

The Binance Chain infrastructure deployment contains so-called "seed" nodes, which have their RPC ports available for access. To find a seed node that is available, you can use the [peers](./dex-api/paths.html#apiv1peers) endpoint to get a list of network peers.

Here is an example of a node that is available for RPC access:

```json
{
   "capabilities" : [
      "node"
   ],
   "listen_addr" : "aa1e4d0d1243a11e9a951063f6065739-7a82be90a58744b6.elb.ap-northeast-1.amazonaws.com:27147",
   "network" : "Binance-Chain-Nile",
   "moniker" : "data-seed-1",
   "access_addr" : "http://aa1e4d0d1243a11e9a951063f6065739-7a82be90a58744b6.elb.ap-northeast-1.amazonaws.com",
   "id" : "data-seed-1",
   "original_listen_addr" : "aa1e4d0d1243a11e9a951063f6065739-7a82be90a58744b6.elb.ap-northeast-1.amazonaws.com:27146",
   "version" : "0.29.1"
}
```

So, using this node, we are able to use raw RPC commands below or the `bnbcli` tool to make a query:

```bash
$ bnbcli dex show -l NNB-0AB_BNB --chain-id Binance-Chain-Nile --node tcp://aa1e4d0d1243a11e9a951063f6065739-7a82be90a58744b6.elb.ap-northeast-1.amazonaws.com:27147
```

