FROM golang:1.11-alpine AS build-env

# Set up dependencies
ENV PACKAGES make git libc-dev bash gcc linux-headers eudev-dev curl ca-certificates ssh

# Set working directory for the build
WORKDIR /go/src/github.com/binance-chain/node

# Add source files
COPY . .

# Add ssh key to download private deps
COPY ~/.ssh/id_rsa /root/ssh/

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apk add --no-cache $PACKAGES && \
    curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh && \
    make get_vendor_deps && \
    make build-alpine && \
    make install

# Final image
FROM alpine:edge

# Install ca-certificates
RUN apk add --update ca-certificates
WORKDIR /root

# Copy over binaries from the build-env
COPY --from=build-env /go/bin/bnbchaind /usr/bin/bnbchaind
COPY --from=build-env /go/bin/bnbcli /usr/bin/bnbcli

# Run gaiad by default, omit entrypoint to ease using container with gaiacli
CMD ["bnbchaind"]
