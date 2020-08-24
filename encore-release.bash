#!/usr/bin/env bash

if [ -z "$GOOS" ]; then
    echo "GOOS must be set"
    exit 1
elif [ -z "$GOARCH" ]; then
    echo "GOARCH must be set"
    exit 1
fi

set -e
set -x

rm -rf encore-build
mkdir encore-build

rm VERSION.cache || true

pushd src
export GOROOT_FINAL=/usr/local/encore/encore-go
./make.bash
popd

rm pkg/tool/${GOOS}_${GOARCH}/{api,oldlink,pprof,trace,cover,objdump,doc,nm} || true
rm bin/gofmt || true
rm -rf pkg/bootstrap || true
rm -rf pkg/obj || true
rm -rf pkg/${GOOS}_${GOARCH}/cmd || true
rm -rf pkg/${GOOS}_${GOARCH}_dynlink || true
rm -rf pkg/${GOOS}_${GOARCH}_shared || true
rm -rf pkg/${GOOS}_${GOARCH}_testcshared_shared || true

cp -r src pkg bin lib LICENSE encore-build/.