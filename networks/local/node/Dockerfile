FROM alpine:3.7

RUN apk update && \
    apk upgrade && \
    apk --no-cache add curl jq file

VOLUME [ /bnbchaind ]
WORKDIR /bnbchaind
EXPOSE 46656 46657
ENTRYPOINT ["/usr/bin/wrapper.sh"]
CMD ["start"]
STOPSIGNAL SIGTERM

COPY wrapper.sh /usr/bin/wrapper.sh