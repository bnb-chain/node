#!/usr/bin/env bash
set -e

# This file downloads all of the binary dependencies we have, and checks out a
# specific git hash.

mkdir -p "vendor/github.com"
cd "vendor/github.com" || exit 1

installFromGithub() {
	repo=$1
	commit=$2
	echo "--> Installing $repo ($commit)..."
	rm -rf "$repo"
	if [ ! -d "$repo" ]; then
		mkdir -p "$repo"
		git clone "https://github.com/$repo.git" "$repo"
		echo githum.com/$repo >> ../modules.txt
	fi
	pushd "$repo" && \
		git fetch origin && \
		git checkout -q "$commit" && \
		rm -rf .git && \
		popd || exit 1
	echo "--> Done"
	echo ""
}

go install github.com/nomad-software/vend@latest
rm -rf vendor
vend
installFromGithub petermattis/goid b0b1615b78e5ee59739545bb38426383b2cda4c9
installFromGithub sasha-s/go-deadlock d68e2bc52ae3291765881b9056f2c1527f245f1e
