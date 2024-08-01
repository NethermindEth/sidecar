
.PHONY: deps proto

deps/go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	go get \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
		google.golang.org/protobuf/cmd/protoc-gen-go \
		google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go install \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
		google.golang.org/protobuf/cmd/protoc-gen-go \
		google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go mod tidy

deps-linux: deps/go
	BIN="/usr/local/bin" VERSION="1.32.2" && \
	curl -sSL "https://github.com/bufbuild/buf/releases/download/v${VERSION}/buf-$(uname -s)-$(uname -m)" -o "${BIN}/buf" && \
	chmod +x "${BIN}/buf"

deps: deps/go
	brew install bufbuild/buf/buf

PROTO_OPTS=--proto_path=protos --go_out=paths=source_relative:protos

proto:
	buf generate protos

.PHONY: clean
clean:
	rm -rf bin || true

.PHONY: build/cmd/server
build/cmd/server:
	go build -o bin/cmd/server cmd/server/main.go

.PHONY: build/cmd/blockSubscriber
build/cmd/blockSubscriber:
	go build -o bin/cmd/blockSubscriber cmd/blockSubscriber/main.go

.PHONY: build/cmd/workers/backfillBlockIndexer
build/cmd/workers/backfillBlockIndexer:
	go build -o bin/cmd/workers/backfillBlockIndexer cmd/workers/backfillBlockIndexer/main.go

.PHONY: build/cmd/workers/transactionLogIndexer
build/cmd/workers/transactionLogIndexer:
	go build -o bin/cmd/workers/transactionLogIndexer cmd/workers/transactionLogIndexer/main.go

.PHONY: build/cmd/workers/contractIndexer
build/cmd/workers/contractIndexer:
	go build -o bin/cmd/workers/contractIndexer cmd/workers/contractIndexer/main.go

.PHONY: build/cmd/workers/restakedStrategiesIndexer
build/cmd/workers/restakedStrategiesIndexer:
	go build -o bin/cmd/workers/restakedStrategiesIndexer cmd/workers/restakedStrategiesIndexer/main.go

.PHONY: build
build: build/cmd/server build/cmd/workers/backfillBlockIndexer build/cmd/workers/transactionLogIndexer build/cmd/blockSubscriber build/cmd/workers/contractIndexer build/cmd/workers/restakedStrategiesIndexer

docker-buildx:
	docker-buildx build --platform linux/amd64 --push -t 767397703211.dkr.ecr.us-east-1.amazonaws.com/blocklake:$(shell date +%s) -t 767397703211.dkr.ecr.us-east-1.amazonaws.com/blocklake:latest .

.PHONY: test
test:
	./scripts/runTests.sh

.PHONY: ci-test
ci-test: deps test
