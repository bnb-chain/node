FROM alpine:3.7

VOLUME [ /bnbchaind ]
WORKDIR /bnbchaind
EXPOSE 26656 26657 6060
ENTRYPOINT ["/bnbchaind/bnbchaind"]
CMD ["start"]
ENV CommitVersion={{COMMIT_VERSION}}
STOPSIGNAL SIGTERM
COPY bnbchaind /bnbchaind
COPY bnbcli /bnbchaind/bnbcli
COPY bnbsentry /bnbchaind/bnbsentry

