#!/usr/bin/env bash
checksum() {
    echo $(sha256sum $@ | awk '{print $1}')
}
change_log_file="./CHANGELOG.md"
version="## $@"
version_prefix="## "
start=0
CHANGE_LOG=""
while read line; do
    if [[ $line == *"$version"* ]]; then
        start=1
        continue
    fi
    if [[ $line == *"$version_prefix"* ]] && [ $start == 1 ]; then
        break;
    fi
    if [ $start == 1 ]; then
        CHANGE_LOG+="$line\n"
    fi
done < ${change_log_file}
MAINNET_ZIP_SUM="$(checksum mainnet_config.zip)"
TESTNET_ZIP_SUM="$(checksum testnet_config.zip)"
LINUX_BIN_SUM="$(checksum linux_binary.zip)"
MAC_BIN_SUM="$(checksum macos_binary.zip)"
WINDOWS_BIN_SUM="$(checksum windows_binary.zip)"
OUTPUT=$(cat <<-END
## Changelog\n
${CHANGE_LOG}\n
## Assets\n
|    Assets    | Sha256 Checksum  |\n
| :-----------: |------------|\n
| mainnet_config.zip | ${MAINNET_ZIP_SUM} |\n
| testnet_config.zip | ${TESTNET_ZIP_SUM} |\n
| linux_binary.zip | ${LINUX_BIN_SUM} |\n
| macos_binary.zip  | ${MAC_BIN_SUM} |\n
| windows_binary.zip  | ${WINDOWS_BIN_SUM} |\n
END
)
echo -e ${OUTPUT}
