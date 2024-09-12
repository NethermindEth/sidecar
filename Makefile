.PHONY: deps proto

args=CGO_ENABLED=1
GO=$(shell which go)

deps/dev:
	${GO} install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
	${GO} install honnef.co/go/tools/cmd/staticcheck@latest
	${GO} install github.com/google/yamlfmt/cmd/yamlfmt@latest

deps/go:
	${GO} install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	${GO} install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	${GO} get \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
		google.golang.org/protobuf/cmd/protoc-gen-go \
		google.golang.org/grpc/cmd/protoc-gen-go-grpc
	${GO} install \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
		google.golang.org/protobuf/cmd/protoc-gen-go \
		google.golang.org/grpc/cmd/protoc-gen-go-grpc
	${GO} mod tidy
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.61.0


deps-linux: deps/go deps/dev
    GOROOT=$(go env GOROOT)
    BIN="${GOROOT}/bin" VERSION="1.32.2" && \
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

.PHONY: build/cmd/sidecar
build/cmd/sidecar:
	${args} ${GO} build -o bin/sidecar main.go

.PHONY: build
build: build/cmd/sidecar

docker-buildx-self:
	docker buildx build -t go-sidecar:latest -t go-sidecar:latest .

docker-buildx:
	docker-buildx build --platform linux/amd64 --push -t 767397703211.dkr.ecr.us-east-1.amazonaws.com/go-sidecar:$(shell date +%s) -t 767397703211.dkr.ecr.us-east-1.amazonaws.com/go-sidecar:latest .

.PHONY: yamlfmt
yamlfmt:
	yamlfmt -lint .github/workflows/*.yml .github/*.yml

.PHONY: fmt
fmt:
	gofmt -l .

.PHONY: fmtcheck
fmtcheck:
	@unformatted_files=$$(gofmt -l .); \
	if [ -n "$$unformatted_files" ]; then \
		echo "The following files are not properly formatted:"; \
		echo "$$unformatted_files"; \
		echo "Please run 'gofmt -w .' to format them."; \
		exit 1; \
	fi
.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	TESTING=true go test -v -p 1 -parallel 1 ./...

.PHONY: staticcheck
staticcheck:
	staticcheck ./...

.PHONY: ci-test
ci-test: test

test-rewards:
	TEST_REWARDS=true TESTING=true ${GO} test ./pkg/rewards -v -p 1

# -----------------------------------------------------------------------------
# SQLite extension build steps
# -----------------------------------------------------------------------------
CC = gcc -g -fPIC -shared

PYTHON_CONFIG = python3-config
PYTHON_VERSION = $(shell python3 -c "import sys; print('{}.{}'.format(sys.version_info.major, sys.version_info.minor))")
PYTHON_LIBDIR := $(shell $(PYTHON_CONFIG) --prefix)/lib

# Base flags
CFLAGS =
LDFLAGS =

INCLUDE_DIRS =
CFLAGS += $(foreach dir,$(INCLUDE_DIRS),-I$(dir))

# Python flags
PYTHON_CFLAGS := $(shell $(PYTHON_CONFIG) --includes)
PYTHON_LDFLAGS := $(shell $(PYTHON_CONFIG) --ldflags)

SQLITE_DIR = /opt/homebrew/opt/sqlite
CFLAGS += -I$(SQLITE_DIR)/include
LDFLAGS += -L$(SQLITE_DIR)/lib -lsqlite3

CFLAGS += $(PYTHON_CFLAGS)
LDFLAGS += $(PYTHON_LDFLAGS) -L$(PYTHON_LIBDIR) -lpython$(PYTHON_VERSION)


.PHONY: sqlite-extensions
sqlite-extensions:
	$(CC) $(CFLAGS) -o sqlite-extensions/libcalculations.dylib sqlite-extensions/calculations.c $(LDFLAGS)
