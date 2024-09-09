## Running

```bash
SIDECAR_DEBUG=false \
SIDECAR_ETHEREUM_RPC_BASE_URL="http://54.198.82.217:8545" \
SIDECAR_ENVIRONMENT="testnet" \
SIDECAR_NETWORK="holesky" \
SIDECAR_ETHERSCAN_API_KEYS="" \
SIDECAR_STATSD_URL="localhost:8125" \
SIDECAR_SQLITE_DB_FILE_PATH="./sqlite/sidecar.db" \
go run cmd/sidecar/main.go
```

## RPC Routes

### Get current block height
```bash
grpcurl -plaintext -d '{}'  localhost:7100 eigenlayer.sidecar.api.v1.Rpc/GetBlockHeight
```
