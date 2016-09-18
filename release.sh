#!/bin/bash

set -e

if [[ ! $(type -P gox) ]]; then
    echo "Error: gox not found."
    echo "To fix: run 'go get github.com/mitchellh/gox', and/or add \$GOPATH/bin to \$PATH"
    exit 1
fi

VER=$1

git tag $VER

echo "Building $VER"
echo

gox -ldflags "-X main.version $VER" -osarch="darwin/amd64 linux/amd64"

echo "$ sha1sum holen_*"
sha1sum holen_*
echo "$ sha256sum holen_*"
sha256sum holen_*
echo "$ md5sum holen_*"
md5sum holen_*
