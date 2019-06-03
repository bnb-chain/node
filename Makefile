GOTOOLS = \
	github.com/mitchellh/gox \
	github.com/golang/dep/cmd/dep \
	github.com/golangci/golangci-lint/cmd/golangci-lint \
	github.com/gogo/protobuf/protoc-gen-gogo \
	github.com/square/certstrap
GOBIN?=${GOPATH}/bin

PACKAGES=$(shell go list ./... | grep -v '/vendor/')
COMMIT_HASH := $(shell git rev-parse --short HEAD)

COSMOS_RELEASE := $(shell grep 'github.com/binance-chain/bnc-cosmos-sdk' Gopkg.toml -n1|grep version|awk '{print $$4}'| sed 's/\"//g')
TENDER_RELEASE := $(shell grep 'github.com/tendermint/tendermint' Gopkg.toml -n2|grep version|awk '{print $$4}'| sed 's/\"//g')

BUILD_TAGS = netgo
BUILD_CLI_TAGS = netgo
BUILD_FLAGS = -tags "${BUILD_TAGS}" -ldflags "-w -s -X github.com/binance-chain/node/version.GitCommit=${COMMIT_HASH} -X github.com/binance-chain/node/version.CosmosRelease=${COSMOS_RELEASE} -X github.com/binance-chain/node/version.TendermintRelease=${TENDER_RELEASE}"
BUILD_CLI_FLAGS = -tags "${BUILD_CLI_TAGS}" -ldflags "-X github.com/binance-chain/node/version.GitCommit=${COMMIT_HASH} -X github.com/binance-chain/node/version.CosmosRelease=${COSMOS_RELEASE} -X github.com/binance-chain/node/version.TendermintRelease=${TENDER_RELEASE}"
# Without -lstdc++ on CentOS we will encounter link error, solution comes from: https://stackoverflow.com/a/29285011/1147187
BUILD_CGOFLAGS = CGO_ENABLED=1 CGO_LDFLAGS="-lleveldb -lsnappy -lstdc++"
BUILD_CFLAGS = ${BUILD_FLAGS} -tags "gcc libsecp256k1"
BUILD_TESTNET_FLAGS = ${BUILD_CLI_FLAGS} -ldflags "-X github.com/binance-chain/node/app.Bech32PrefixAccAddr=tbnb"

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    # On CentOS, this requires on control machine:
    # yum group install "Development Tools"
    # yum install glibc-static
    # build cmake, leveldb, snappy and copy libleveldb.a and libsnappy.a into /usr/lib
	BUILD_CFLAGS += -ldflags '-extldflags "-static"'
endif

all: get_vendor_deps format build

LEDGER_ENABLED ?= true

########################################
### Build/Install

ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      BUILD_CLI_TAGS += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        BUILD_CLI_TAGS += ledger
      endif
    endif
  endif
endif

########################################
### CI

ci: get_vendor_deps build

########################################
### Build

define buildwindows
     go build $(BUILD_CLI_FLAGS) -o build/bnbcli.exe ./cmd/bnbcli
     go build $(BUILD_TESTNET_FLAGS) -o build/tbnbcli.exe ./cmd/bnbcli
     go build $(BUILD_FLAGS) -o build/bnbchaind.exe ./cmd/bnbchaind
     go build $(BUILD_FLAGS) -o build/bnbsentry.exe ./cmd/bnbsentry
     go build $(BUILD_FLAGS) -o build/pressuremaker.exe ./cmd/pressuremaker
     go build $(BUILD_FLAGS) -o build/lightd.exe ./cmd/lightd
endef


build: get_tools
ifeq ($(OS),Windows_NT)
	$(call buildwindows)
else
	go build $(BUILD_CLI_FLAGS) -o build/bnbcli ./cmd/bnbcli
	go build $(BUILD_TESTNET_FLAGS) -o build/tbnbcli ./cmd/bnbcli
	go build $(BUILD_FLAGS) -o build/bnbchaind ./cmd/bnbchaind
	go build $(BUILD_FLAGS) -o build/bnbsentry ./cmd/bnbsentry
	go build $(BUILD_FLAGS) -o build/pressuremaker ./cmd/pressuremaker
	go build $(BUILD_FLAGS) -o build/lightd ./cmd/lightd
endif

build_c:
ifeq ($(OS),Windows_NT)
	go build $(BUILD_CLI_FLAGS) -o build/bnbcli.exe ./cmd/bnbcli
	go build $(BUILD_TESTNET_FLAGS) -o build/tbnbcli.exe ./cmd/bnbcli
	$(BUILD_CGOFLAGS) go build $(BUILD_CFLAGS) -o build/bnbchaind.exe ./cmd/bnbchaind
	$(BUILD_CGOFLAGS) go build $(BUILD_CFLAGS) -o build/bnbsentry.exe ./cmd/bnbsentry
	go build $(BUILD_FLAGS) -o build/pressuremaker.exe ./cmd/pressuremaker
	$(BUILD_CGOFLAGS) go build $(BUILD_CFLAGS) -o build/lightd.exe ./cmd/lightd
else
	go build $(BUILD_CLI_FLAGS) -o build/bnbcli ./cmd/bnbcli
	go build $(BUILD_TESTNET_FLAGS) -o build/tbnbcli ./cmd/bnbcli
	$(BUILD_CGOFLAGS) go build $(BUILD_CFLAGS) -o build/bnbchaind ./cmd/bnbchaind
	$(BUILD_CGOFLAGS) go build $(BUILD_CFLAGS) -o build/bnbsentry ./cmd/bnbsentry
	go build $(BUILD_FLAGS) -o build/pressuremaker ./cmd/pressuremaker
	$(BUILD_CGOFLAGS) go build $(BUILD_CFLAGS) -o build/lightd ./cmd/lightd
endif

build-windows:
	$(call buildwindows)

build-linux:
	LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 $(MAKE) build

build-linux_c:
	LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 $(MAKE) build_c

build-alpine:
	LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(MAKE) build

build-alpine_c:
    LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(MAKE) build_c

install:
	go install $(BUILD_FLAGS) ./cmd/bnbchaind
	go install $(BUILD_CLI_FLAGS) ./cmd/bnbcli
	go install $(BUILD_FLAGS) ./cmd/bnbsentry

install_c:
	$(BUILD_CGOFLAGS) go install $(BUILD_CFLAGS) ./cmd/bnbchaind
	go install $(BUILD_CLI_FLAGS) ./cmd/bnbcli
	go install $(BUILD_FLAGS) ./cmd/bnbsentry

########################################
### Dependencies

get_vendor_deps:
	@rm -rf vendor/
	@echo "--> Running dep ensure"
	@dep ensure -v
	@go get golang.org/x/tools/cmd/goimports

########################################
### Format
format:
	@echo "-->Formatting"
	$(shell cd ../../../ && goimports -w -local github.com/binance-chain/node $(PACKAGES))
	$(shell cd ../../../ && gofmt -w $(PACKAGES))

########################################
### Lint
lint:
	@echo "-->Lint"
	golint $(PACKAGES)

########################################
### Testing

get_tools:
	@echo "--> Installing tools"
	./scripts/get_tools.sh

test:
	make set_with_deadlock
	make test_unit
	make test_race
	make cleanup_after_test_with_deadlock

# uses https://github.com/sasha-s/go-deadlock/ to detect potential deadlocks
set_with_deadlock:
	find . -name "*.go" | grep -v "vendor/" | xargs -n 1 sed -i.mutex_bak 's/sync.RWMutex/deadlock.RWMutex/'
	find . -name "*.go" | grep -v "vendor/" | xargs -n 1 sed -i.mutex_bak 's/sync.Mutex/deadlock.Mutex/'
	find . -name "*.go" | grep -v "vendor/" | xargs -n 1 goimports -w

# cleanes up after you ran test_with_deadlock
cleanup_after_test_with_deadlock:
	find . -name "*.go" | grep -v "vendor/" | xargs -n 1 sed -i.mutex_bak 's/deadlock.RWMutex/sync.RWMutex/'
	find . -name "*.go" | grep -v "vendor/" | xargs -n 1 sed -i.mutex_bak 's/deadlock.Mutex/sync.Mutex/'
	find . -name "*.go" | grep -v "vendor/" | xargs -n 1 goimports -w
	find . -name "*.go.mutex_bak" | grep -v "vendor/" | xargs rm


test_race:
	@go test -race $(PACKAGES)

test_unit:
	@go test $(PACKAGES)

integration_test: build
	@echo "-->Integration Test"
	@./integration_test.sh

########################################
### Pre Commit
pre_commit: build test format

########################################
### Local validator nodes using docker and docker-compose
build-docker-node:
	$(MAKE) -C networks/local

# Run a 4-node testnet locally
localnet-start: localnet-stop
	@if ! [ -f build/node0/gaiad/config/genesis.json ]; then docker run --rm -v $(CURDIR)/build:/bnbchaind:Z binance/bnbdnode testnet --v 4 -o . --starting-ip-address 172.20.0.2 ; fi
	@for i in `seq 0 3`; do \
		if [ "$(SKIP_TIMEOUT)" = "true" ]; then \
			sed -i -e "s/skip_timeout_commit = false/skip_timeout_commit = true/g" ./build/node$$i/gaiad/config/config.toml;\
		else \
			sed -i -e "s/skip_timeout_commit = true/skip_timeout_commit = false/g" ./build/node$$i/gaiad/config/config.toml;\
		fi;\
	done
	@for i in `seq 0 3`; do \
		if [ "$(PEX)" = "false" ]; then \
			sed -i -e "s/pex = true/pex = false/g" ./build/node$$i/gaiad/config/config.toml;\
		else \
			sed -i -e "s/pex = false/pex = true/g" ./build/node$$i/gaiad/config/config.toml;\
		fi;\
	done
	docker-compose up

# Stop testnet
localnet-stop:
	docker-compose down

# To avoid unintended conflicts with file names, always add to .PHONY
# unless there is a reason not to.
# https://www.gnu.org/software/make/manual/html_node/Phony-Targets.html
.PHONY: build install get_vendor_deps test test_unit build-linux build-docker-node localnet-start localnet-stop
