PACKAGES=$(shell go list ./... | grep -v '/vendor/')
COMMIT_HASH := $(shell git rev-parse --short HEAD)
BUILD_FLAGS = -ldflags "-X github.com/BiJie/BinanceChain/version.GitCommit=${COMMIT_HASH}"

all: get_vendor_deps build

########################################
### CI

ci: get_tools get_vendor_deps build test_cover

########################################
### Build

build: format
ifeq ($(OS),Windows_NT)
	go build $(BUILD_FLAGS) -o build/bnbcli.exe ./cmd/bnbcli
	go build $(BUILD_FLAGS) -o build/bnbchaind.exe ./cmd/bnbchaind
else
	go build $(BUILD_FLAGS) -o build/bnbcli ./cmd/bnbcli
	go build $(BUILD_FLAGS) -o build/bnbchaind ./cmd/bnbchaind
endif

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

# To avoid unintended conflicts with file names, always add to .PHONY
# unless there is a reason not to.
# https://www.gnu.org/software/make/manual/html_node/Phony-Targets.html
.PHONY: build install get_vendor_deps test test_unit
