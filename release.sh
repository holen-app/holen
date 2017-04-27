#!/bin/bash

set -e

if [[ ! $(type -P gox) ]]; then
    echo "Error: gox not found."
    echo "To fix: run 'go get github.com/mitchellh/gox', and/or add \$GOPATH/bin to \$PATH"
    exit 1
fi

if [[ ! $(type -P github-release) ]]; then
    echo "Error: github-release not found."
    exit 1
fi

VER=$1

if [[ -z $VER ]]; then
    echo "Need to specify version."
    exit 1
fi

PRE_ARG=
if [[ $VER =~ pre ]]; then
    PRE_ARG="--pre-release"
fi

git tag $VER

echo "Building $VER"
echo

gox -ldflags "-X main.version=$VER" -osarch="darwin/amd64 linux/amd64 windows/amd64 linux/arm"

echo "* " > desc
echo "" >> desc

echo "$ sha1sum holen_*" >> desc
sha1sum holen_* >> desc
echo "$ sha256sum holen_*" >> desc
sha256sum holen_* >> desc
echo "$ md5sum holen_*" >> desc
md5sum holen_* >> desc

vi desc

cp bootstrap.template holen.bootstrap
perl -p -i -e "s/__VERSION__/$VER/g" holen.bootstrap
md5sum holen_* >> holen.bootstrap

git push --tags

sleep 2

cat desc | github-release release $PRE_ARG --user holen-app --repo holen --tag $VER --name $VER --description -
github-release upload --user holen-app --repo holen --tag $VER --name holen_darwin_amd64 --file holen_darwin_amd64
github-release upload --user holen-app --repo holen --tag $VER --name holen_linux_amd64 --file holen_linux_amd64
github-release upload --user holen-app --repo holen --tag $VER --name holen_linux_arm --file holen_linux_arm
github-release upload --user holen-app --repo holen --tag $VER --name holen_windows_amd64.exe --file holen_windows_amd64.exe
github-release upload --user holen-app --repo holen --tag $VER --name holen.bootstrap --file holen.bootstrap
