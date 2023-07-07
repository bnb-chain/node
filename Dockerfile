FROM golang:1.19-alpine AS build-env

# Set up dependencies
ENV PACKAGES make cmake git libc-dev bash gcc linux-headers eudev-dev curl ca-certificates build-base libc-dev
ENV CGO_CFLAGS="-O -D__BLST_PORTABLE__"
ENV CGO_CFLAGS_ALLOW="-O -D__BLST_PORTABLE__"

# Set working directory for the build
WORKDIR /go/src/github.com/bnb-chain/node

# Add source files
COPY . .

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apk add --no-cache $PACKAGES && \
    make build && \
    make install

# # Final image
FROM alpine:3.16.0

# Install dependencies
RUN apk add --update ca-certificates tini bash gcc

ARG USER=bnbchain
ARG USER_UID=1000
ARG USER_GID=1000

ENV DEFAULT_CONFIG=/configs
ENV HOME=/data

RUN addgroup -g ${USER_GID} ${USER} \
  && adduser -u ${USER_UID} -G ${USER} --shell /sbin/nologin --no-create-home -D ${USER} \
  && addgroup ${USER} tty
RUN mkdir -p ${HOME} ${DEFAULT_CONFIG} 
WORKDIR ${HOME}

# Copy over binaries from the build-env
COPY --from=build-env /go/bin/bnbchaind /usr/bin/bnbchaind
COPY --from=build-env /go/bin/bnbcli /usr/bin/bnbcli
COPY docker-entrypoint.sh /
COPY ./asset/ ${DEFAULT_CONFIG}/

RUN chown -R ${USER_UID}:${USER_GID} ${HOME} \
  && chmod +x /docker-entrypoint.sh

USER ${USER}:${USER}

# Run gaiad by default, omit entrypoint to ease using container with gaiacli
CMD ["/sbin/tini", "--", "/docker-entrypoint.sh"]
