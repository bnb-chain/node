FROM alpine:3.7

VOLUME [ /bnbchaind ]
WORKDIR /bnbchaind
EXPOSE 26656 26657 6060
ENTRYPOINT ["/bnbchaind/bnbchaind"]
CMD ["start"]
STOPSIGNAL SIGTERM
COPY bnbchaind /bnbchaind
COPY bnbcli /bnbchaind/bnbcli

