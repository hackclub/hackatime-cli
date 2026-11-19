#!/bin/bash

set -e

# ensure existence of release folder
if ! [ -d "./release" ]; then
    mkdir ./release
fi

# ensure zip is installed
if [ "$(which zip)" = "" ]; then
    apt-get update && apt-get install -y zip
fi

# add execution permission
chmod 750 ./build/hackatime-cli-android-arm
chmod 750 ./build/hackatime-cli-android-arm64
chmod 750 ./build/hackatime-cli-darwin-amd64
chmod 750 ./build/hackatime-cli-darwin-arm64
chmod 750 ./build/hackatime-cli-freebsd-386
chmod 750 ./build/hackatime-cli-freebsd-amd64
chmod 750 ./build/hackatime-cli-freebsd-arm
chmod 750 ./build/hackatime-cli-linux-386
chmod 750 ./build/hackatime-cli-linux-amd64
chmod 750 ./build/hackatime-cli-linux-arm
chmod 750 ./build/hackatime-cli-linux-arm64
chmod 750 ./build/hackatime-cli-linux-riscv64
chmod 750 ./build/hackatime-cli-netbsd-386
chmod 750 ./build/hackatime-cli-netbsd-amd64
chmod 750 ./build/hackatime-cli-netbsd-arm
chmod 750 ./build/hackatime-cli-openbsd-386
chmod 750 ./build/hackatime-cli-openbsd-amd64
chmod 750 ./build/hackatime-cli-openbsd-arm
chmod 750 ./build/hackatime-cli-openbsd-arm64
chmod 750 ./build/hackatime-cli-windows-386.exe
chmod 750 ./build/hackatime-cli-windows-amd64.exe
chmod 750 ./build/hackatime-cli-windows-arm64.exe

# create archives
zip -j ./release/hackatime-cli-android-arm.zip ./build/hackatime-cli-android-arm
zip -j ./release/hackatime-cli-android-arm64.zip ./build/hackatime-cli-android-arm64
zip -j ./release/hackatime-cli-darwin-amd64.zip ./build/hackatime-cli-darwin-amd64
zip -j ./release/hackatime-cli-darwin-arm64.zip ./build/hackatime-cli-darwin-arm64
zip -j ./release/hackatime-cli-freebsd-386.zip ./build/hackatime-cli-freebsd-386
zip -j ./release/hackatime-cli-freebsd-amd64.zip ./build/hackatime-cli-freebsd-amd64
zip -j ./release/hackatime-cli-freebsd-arm.zip ./build/hackatime-cli-freebsd-arm
zip -j ./release/hackatime-cli-linux-386.zip ./build/hackatime-cli-linux-386
zip -j ./release/hackatime-cli-linux-amd64.zip ./build/hackatime-cli-linux-amd64
zip -j ./release/hackatime-cli-linux-arm.zip ./build/hackatime-cli-linux-arm
zip -j ./release/hackatime-cli-linux-arm64.zip ./build/hackatime-cli-linux-arm64
zip -j ./release/hackatime-cli-linux-riscv64.zip ./build/hackatime-cli-linux-riscv64
zip -j ./release/hackatime-cli-netbsd-386.zip ./build/hackatime-cli-netbsd-386
zip -j ./release/hackatime-cli-netbsd-amd64.zip ./build/hackatime-cli-netbsd-amd64
zip -j ./release/hackatime-cli-netbsd-arm.zip ./build/hackatime-cli-netbsd-arm
zip -j ./release/hackatime-cli-openbsd-386.zip ./build/hackatime-cli-openbsd-386
zip -j ./release/hackatime-cli-openbsd-amd64.zip ./build/hackatime-cli-openbsd-amd64
zip -j ./release/hackatime-cli-openbsd-arm.zip ./build/hackatime-cli-openbsd-arm
zip -j ./release/hackatime-cli-openbsd-arm64.zip ./build/hackatime-cli-openbsd-arm64
zip -j ./release/hackatime-cli-windows-386.zip ./build/hackatime-cli-windows-386.exe
zip -j ./release/hackatime-cli-windows-amd64.zip ./build/hackatime-cli-windows-amd64.exe
zip -j ./release/hackatime-cli-windows-arm64.zip ./build/hackatime-cli-windows-arm64.exe

# calculate checksums
for file in  ./release/*; do
	checksum=$(sha256sum "${file}" | cut -d' ' -f1)
	filename=$(echo "${file}" | rev | cut -d/ -f1 | rev)
	echo "${checksum} ${filename}" >> ./release/checksums_sha256.txt
done
