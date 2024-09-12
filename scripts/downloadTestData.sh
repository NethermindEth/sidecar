#!/usr/bin/env bash

version=$(cat .testdataVersion)
bucketName="eigenlayer-sidecar-testdata"

dataUrl="https://${bucketName}.s3.amazonaws.com/${version}.tar"

if [[ -z $version ]]; then
  echo "No version found in .testdataVersion"
  exit 1
fi
echo "Downloading testdata version $dataUrl"
curl -L $dataUrl | tar xvz -C ./
