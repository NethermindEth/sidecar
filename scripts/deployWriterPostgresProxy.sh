#!/usr/bin/env bash

kubectl run postgres-proxy-write \
  --image docker.io/alpine/socat \
  --namespace blocklake-dev \
  -- \
  tcp-listen:5432,fork,reuseaddr \
  tcp-connect:blocklake.cluster-cjg0ui0ksnx8.us-east-1.rds.amazonaws.com:5432
