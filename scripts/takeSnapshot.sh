#!/usr/bin/env bash

DBNAME=$1
NETWORK=$2
SCHEMA=$3
DESTINATION=$4

_usage="Usage: $0 <dbname> <network> <schema> <destination>"

if [[ -z $DBNAME ]]; then
    echo $_usage
    exit 1
fi

if [[ -z $NETWORK ]]; then
    echo $_usage
    exit 1
fi

networks=("mainnet" "testnet" "preprod")
if [[ ! " ${networks[@]} " =~ " ${NETWORK} " ]]; then
    echo "Invalid network"
    exit 1
fi

if [[ -z $SCHEMA ]]; then
    echo $_usage
    exit 1
fi

if [[ -z $DESTINATION ]]; then
    echo $_usage
    exit 1
fi

echo "Snapshotting database $DBNAME"

pg_dump --schema $SCHEMA -Fc $DBNAME > $DESTINATION

echo "Done!"
