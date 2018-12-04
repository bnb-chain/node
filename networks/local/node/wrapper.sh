#!/usr/bin/env sh

##
## Input parameters
##
BINARY=/bnbchaind/${BINARY:-bnbchaind}
ID=${ID:-0}
LOG=${LOG:-bnbchaind.log}

##
## Assert linux binary
##
if ! [ -f "${BINARY}" ]; then
	echo "The binary $(basename "${BINARY}") cannot be found. Please add the binary to the shared folder. Please use the BINARY environment variable if the name of the binary is not 'gaiad' E.g.: -e BINARY=gaiad_my_test_version"
	exit 1
fi
BINARY_CHECK="$(file "$BINARY" | grep 'ELF 64-bit LSB executable, x86-64')"
if [ -z "${BINARY_CHECK}" ]; then
	echo "Binary needs to be OS linux, ARCH amd64"
	exit 1
fi

##
## Run binary with all parameters
##
export BCHOME="/bnbchaind/node${ID}/gaiad"

if [ -d "`dirname ${BCHOME}/${LOG}`" ]; then
  "$BINARY" --home "$BCHOME" "$@" | tee "${BCHOME}/${LOG}"
else
  "$BINARY" --home "$BCHOME" "$@"
fi

chmod 777 -R /bnbchaind
