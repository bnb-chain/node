FROM golang:1.17-alpine AS build-env

# Set up dependencies
ENV PACKAGES make git libc-dev bash gcc linux-headers eudev-dev curl ca-certificates

# Set working directory for the build
WORKDIR /bnb-chain/node

# Add source files
COPY . /bnb-chain/node

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apk update && \
    apk add --update --no-cache $PACKAGES

RUN make build

# Final image
FROM alpine:edge

# Install ca-certificates
RUN apk add --update ca-certificates
WORKDIR /root

# Copy over binaries from the build-env
COPY --from=build-env /bnb-chain/node/build/bnbchaind /usr/bin/bnbchaind
COPY --from=build-env /bnb-chain/node/build/bnbcli /usr/bin/bnbcli

# Run gaiad by default, omit entrypoint to ease using container with gaiacli
CMD ["bnbchaind"]
