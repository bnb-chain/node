#!/usr/bin/env bash
set -e

# This file downloads all of the binary dependencies we have, and checks out a
# specific git hash.

# check if GOPATH is set
#if [ -z ${GOPATH+x} ]; then
#	echo "please set GOPATH (https://github.com/golang/go/wiki/SettingGOPATH)"
#	exit 1
#fi

mkdir -p "vendor/github.com"
cd "vendor/github.com" || exit 1

installFromGithub() {
	repo=$1
	commit=$2
	# optional
#	subdir=$3
	echo "--> Installing $repo ($commit)..."
	rm -rf "$repo"
	if [ ! -d "$repo" ]; then
		mkdir -p "$repo"
		git clone "https://github.com/$repo.git" "$repo"
		echo githum.com/$repo >> ../modules.txt
	fi
#	if [ ! -z ${subdir+x} ] && [ ! -d "$repo/$subdir" ]; then
#		echo "ERROR: no such directory $repo/$subdir"
#		exit 1
#	fi
	pushd "$repo" && \
		git fetch origin && \
		git checkout -q "$commit" && \
		rm -rf .git && \
#		if [ ! -z ${subdir+x} ]; then cd "$subdir" || exit 1; fi && \
#		go install && \
#		if [ ! -z ${subdir+x} ]; then cd - || exit 1; fi && \
		popd || exit 1
	echo "--> Done"
	echo ""
}

######################## COMMON TOOLS ########################################
## XXX: https://github.com/tendermint/tendermint/issues/3242
installFromGithub petermattis/goid b0b1615b78e5ee59739545bb38426383b2cda4c9
installFromGithub sasha-s/go-deadlock d68e2bc52ae3291765881b9056f2c1527f245f1e
#go get github.com/petermattis/goid@b0b1615b78e5ee59739545bb38426383b2cda4c9
#go get github.com/sasha-s/go-deadlock@d68e2bc52ae3291765881b9056f2c1527f245f1e
