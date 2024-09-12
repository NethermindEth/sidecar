FROM golang:1.22-bullseye as build

RUN apt-get update
RUN apt-get install -y make

RUN mkdir /build

COPY . /build

WORKDIR /build

RUN make deps-linux

RUN make build

FROM debian:stable-slim as run

RUN apt-get update
RUN apt-get install -y ca-certificates

COPY --from=build /build /build
