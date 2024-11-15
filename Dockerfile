FROM debian:testing-slim AS builder

ARG TARGETARCH

RUN apt update && \
    apt install -y make curl git

WORKDIR /build

COPY . .

# system and linux dependencies
RUN make deps

RUN make build

FROM debian:testing-slim

RUN apt-get update && apt-get install -y ca-certificates

COPY --from=builder /build/bin/sidecar /usr/local/bin/sidecar

ENTRYPOINT ["/usr/local/bin/sidecar"]
