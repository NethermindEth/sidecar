FROM debian:stable-slim AS builder

ARG TARGETARCH

RUN export DEBIAN_FRONTEND=noninteractive && \
    apt update && \
    apt install -y -q --no-install-recommends \
        make \
        linux-headers-${TARGETARCH} && \
    apt clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY . .

# system and linux dependencies
RUN make deps

RUN make build

ENTRYPOINT ["/usr/local/bin/sidecar"]
