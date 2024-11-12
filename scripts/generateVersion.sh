#!/usr/bin/env bash

if [[ $REF == refs/tags/* ]]; then
    echo -n "${REF#refs/tags/}" > .release_version
else
    v=$(git rev-parse --short HEAD) && echo -n $v > .release_version
fi
