#!/usr/bin/env bash

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -a | tr '[:upper:]' '[:lower:]')

command_exists() {
    command -v "$@" > /dev/null 2>&1
}

apt_update_and_install() {
    if command_exists sudo; then
        sudo apt-get update
        sudo apt-get install "$@"
    else
        apt-get update
        apt-get install "$@"
    fi
}

if [[ "$OS" == "linux" ]]; then
    apt_update_and_install -y \
        make \
        curl \
        git \
        software-properties-common \
        jq

    which go
    if [[ $? != 0 ]]; then
        echo "Installing Go 1.22"
        apt-get install -y golang
    fi
fi
