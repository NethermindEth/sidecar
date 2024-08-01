FROM golang:1.22-bullseye as build

RUN apt-get update
RUN apt-get install -y make postgresql-client

RUN mkdir /build

COPY . /build

WORKDIR /build

RUN make deps-linux

RUN make build

# FROM golang:1.22-bullseye as run
#
# RUN apt-get update
# RUN apt-get install -y vim postgresql-client
#
# COPY --from=build /build/bin/cmd /bin
