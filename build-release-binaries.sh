#!/usr/bin/env sh

set -ex

output_dir=docker-spk-binaries

for os in darwin linux; do
	export GOOS=$os
	export GOARCH=amd64
	mkdir -p $output_dir/$GOOS/$GOARCH
	CGO_ENABLED=0 go build -ldflags '-w -s' -o $output_dir/$GOOS/$GOARCH/docker-spk
done

( cd $output_dir && sha256sum */*/* > sha256sums.txt )
tar -cvvf $output_dir.tar $output_dir
gzip $output_dir.tar
