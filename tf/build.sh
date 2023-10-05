#!/bin/bash

set -e 
set -o pipefail

build () {
## Some instructions to make zip files always the same
## https://zerostride.medium.com/building-deterministic-zip-files-with-built-in-commands-741275116a19
export CGO_ENABLED=0 
export GOOS=linux 
export GOARCH=amd64
export GOPATH=/home/runner/go

echo Building $1
(
    cd $1;
    go build -trimpath -tags lambda.norpc -buildvcs=false -compiler gc -o package/bootstrap;
    chmod 777 package/bootstrap
    echo Binary MD5 $(cat package/bootstrap | md5sum )
    touch package/bootstrap -t 201301250000
)
(
    cd $1/package;
    rm -f ../package.zip;
    zip -rq -D -X -9 -A --compression-method deflate ../package.zip bootstrap;
    echo Zip MD5 $(cat ../package.zip | md5sum )
)
}

go env

build cmd/lambda
build cmd/auditLambda
