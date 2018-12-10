FROM alpine:3.7

VOLUME [ /bnbchaind ]
WORKDIR /bnbchaind
EXPOSE 26656 26657 6060
ENTRYPOINT ["/bnbchaind/bnbchaind"]
CMD ["start"]
ENV CommitVersion=9109a11a94b87167086ad9b5f48cdab725b8b4b2
STOPSIGNAL SIGTERM
COPY bnbchaind /bnbchaind
COPY bnbcli /bnbchaind/bnbcli
COPY bnbsentry /bnbchaind/bnbsentry

