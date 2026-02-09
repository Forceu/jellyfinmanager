#!/bin/bash

set -e

targets=${@-"darwin/amd64 darwin/arm64 linux/amd64 linux/386 linux/arm linux/arm64 linux/riscv64 windows/amd64 windows/arm64"}

cd /usr/src/myapp
go generate ./...

for target in $targets; do
	os="$(echo $target | cut -d '/' -f1)"
	arch="$(echo $target | cut -d '/' -f2)"
	output="build/jellyfinmanager-${os}_${arch}"
	if [ $os = "windows" ]; then
		output+='.exe'
	fi

	echo "----> Building jellyfinmanager for $target"
	GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -ldflags="-s -w" -o $output github.com/forceu/jellyfinmanager
	zip -j $output.zip $output >/dev/null
	rm $output
done

echo "----> Build is complete. List of files at build/:"
cd build/
ls -l jellyfinmanager-*
