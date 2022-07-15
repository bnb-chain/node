## Docker Usage

### Image
```sh
docker pull ghcr.io/bnb-chain/node:latest
```

### Env

| env | desc | default|
|---|---|---|
| NETWORK | default network options, if `mainnet` or `testnet` is configured, the genesis file will be automatically configured | `mainnet`|
| HOME | directory for config and data | `/data` |

### Example
1. Start a testnet full node
```
docker run -p 27146:27146 -p 27147:27147 -e NETWORK=testnet ghcr.io/bnb-chain/node:latest
```

2. Start a mainnet full node with mounted volume
```
docker run -p 27146:27146 -p 27147:27147 -v /tmp/chain/data:/data ghcr.io/bnb-chain/node:latest
```



