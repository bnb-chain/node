PACKAGES=$(shell go list ./... | grep -v '/vendor/')
COMMIT_HASH := $(shell git rev-parse --short HEAD)
BUILD_TAGS = netgo
BUILD_FLAGS = -tags "${BUILD_TAGS}" -ldflags "-X github.com/BiJie/BinanceChain/version.GitCommit=${COMMIT_HASH}"

all: get_vendor_deps format build

########################################
### CI

ci: get_tools get_vendor_deps build test_cover

########################################
### Build

build:
ifeq ($(OS),Windows_NT)
	go build $(BUILD_FLAGS) -o build/bnbcli.exe ./cmd/bnbcli
	go build $(BUILD_FLAGS) -o build/bnbchaind.exe ./cmd/bnbchaind
else
	go build $(BUILD_FLAGS) -o build/bnbcli ./cmd/bnbcli
	go build $(BUILD_FLAGS) -o build/bnbchaind ./cmd/bnbchaind
endif

build-linux:
	LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 $(MAKE) build

install:
	go install $(BUILD_FLAGS) ./cmd/bnbchaind
	go install $(BUILD_FLAGS) ./cmd/bnbcli

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
	$(shell cd ../../../ && goimports -w -local github.com/BiJie/BinanceChain $(PACKAGES))
	$(shell cd ../../../ && gofmt -w $(PACKAGES))

########################################
### Lint
lint:
	@echo "-->Lint"
	golint $(PACKAGES)

########################################
### Testing

test: test_unit

test_unit:
	@go test $(PACKAGES)

########################################
### Pre Commit
pre_commit: build test format

########################################
### Local validator nodes using docker and docker-compose
build-docker-node:
	$(MAKE) -C networks/local

# Run a 4-node testnet locally
localnet-start: localnet-stop
	@if ! [ -f build/node0/gaiad/config/genesis.json ]; then docker run --rm -v $(CURDIR)/build:/bnbchaind:Z binance/bnbdnode testnet --v 4 --o . --starting-ip-address 192.168.10.2 ; fi
	docker-compose up

# Stop testnet
localnet-stop:
	docker-compose down

# To avoid unintended conflicts with file names, always add to .PHONY
# unless there is a reason not to.
# https://www.gnu.org/software/make/manual/html_node/Phony-Targets.html
.PHONY: build install get_vendor_deps test test_unit build-linux build-docker-node localnet-start localnet-stop
