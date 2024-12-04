#!/usr/bin/env bash

DATABASE=$1
NETWORK=$2

if [[ -z $DATABASE ]]; then
    echo "Usage: $0 <database> <network>"
    exit 1
fi

if [[ -z $NETWORK ]]; then
    echo "Usage: $0 <database> <network>"
    exit 1
fi

networks=("mainnet" "testnet" "preprod")
if [[ ! " ${networks[@]} " =~ " ${NETWORK} " ]]; then
    echo "Invalid network"
    exit 1
fi

dropdb $DATABASE || true
createdb $DATABASE

go run main.go database \
    --ethereum.rpc-url="https://ethereum-holesky-rpc.publicnode.com" \
    --chain=$NETWORK \
    --statsd.url="localhost:8125" \
    --database.host="localhost" \
    --database.port="5432" \
    --database.user=$PG_USER \
    --database.password=$PG_PASSWORD \
    --database.db_name=$DATABASE

sourcesPath="snapshots/$NETWORK/sources"

for i in $(ls $sourcesPath | grep '.sql'); do
    full_file="${sourcesPath}/$i"
    echo $full_file
    psql --dbname $DATABASE < $full_file
done

goldFile="$(pwd)/snapshots/${NETWORK}/sources/sidecar-${NETWORK}-holesky-gold.csv"
echo "Loading gold table from: $goldFile"

psql -c "\copy gold_table from '${goldFile}' CSV HEADER;" --dbname $DATABASE

psql -c "insert into state_roots (eth_block_number, eth_block_hash, state_root) select number as eth_block_number, hash as eth_block_hash, 'first state root' as state_root from blocks where number = (select max(number) from blocks);" --dbname $DATABASE


