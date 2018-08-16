# Set up testnet 

## Usage

go to machine `172.31.4.208` and run `./deploy.sh`

```
Options:
 --build     			(default false)build bnbchaind before deploy
 --skip_timeout     	(default true)make progress as soon as we have all the precommits
```

eg
```
./deploy.sh --build true --skip_timeout true
```

log file's path is `~/node*.log` and node's home path is `~/node*`

## Introduction

After execute `deploy.sh`, you will get a testnet like:

![](./testnet.png)

