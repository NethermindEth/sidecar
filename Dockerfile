FROM debian:testing-slim AS builder

ARG TARGETARCH

RUN apt update && \
    apt install -y make curl git

WORKDIR /build

COPY . .

# system and linux dependencies
RUN make deps

RUN make build

RUN mv /build/bin/sidecar /usr/local/bin/sidecar

RUN  apt clean && \
    rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/usr/local/bin/sidecar"]
