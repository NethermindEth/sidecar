## Running

### Directly using Go

*Dependencies*

* Go 1.22
* gRPCurl (for testing)

```bash
# Create the directory to hold the sqlite database
mkdir ./sidecar-data || true

go run main.go run \
    --ethereum.rpc-url="http://34.229.43.36:8545" \
    --chain="holesky" \
    --etherscan.api-keys="<your etherscan key>" \
    --statsd.url="localhost:8125" \
    --datadir="./sidecar-data"

```

### Using the public Docker container
```bash
# Create the directory to hold the sqlite database
mkdir ./sqlite || true

docker run -it --rm \
  -e SIDECAR_DEBUG=false \
  -e SIDECAR_ETHEREUM_RPC_BASE_URL="http://34.229.43.36:8545" \
  -e SIDECAR_CHAIN="holesky" \
  -e SIDECAR_ETHERSCAN_API_KEYS="<your etherscan key>" \
  -e SIDECAR_STATSD_URL="localhost:8125" \
  -e SIDECAR_DATADIR="/sidecar" \
  -v "$(pwd)/sqlite:/sidecar" \
  --tty -i \
  public.ecr.aws/z6g0f8n7/go-sidecar:latest run
```

### Build and run a container locally
```bash
# Create the directory to hold the sqlite database
mkdir ./sqlite || true

make docker-buildx-self

docker run \
  -e "SIDECAR_DEBUG=false" \
  -e "SIDECAR_ETHEREUM_RPC_BASE_URL=http://34.229.43.36:8545" \
  -e "SIDECAR_CHAIN=holesky" \
  -e "SIDECAR_ETHERSCAN_API_KEYS='<your etherscan key>'" \
  -e "SIDECAR_STATSD_URL=localhost:8125" \
  -e SIDECAR_DATADIR="/sidecar" \
  -v "$(pwd)/sqlite:/sidecar" \
  --tty -i \
  go-sidecar:latest run
```

## RPC Routes

### Get current block height

```bash
grpcurl -plaintext -d '{}'  localhost:7100 eigenlayer.sidecar.api.v1.Rpc/GetBlockHeight
```

### Get the stateroot at a block height

```bash
grpcurl -plaintext -d '{ "blockNumber": 1140438 }'  localhost:7100 eigenlayer.sidecar.api.v1.Rpc/GetStateRoot
