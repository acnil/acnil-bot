#!/bin/bash

set -e 
set -o pipefail

export CGO_ENABLED=0 
export GOOS=linux 
export GOARCH=amd64
export GOPATH=/home/runner/go

build="go build -trimpath -tags lambda.norpc -buildvcs=true -compiler gc"
go env

$build -o cmd/lambda/package/bootstrap cmd/lambda/main.go

chmod 777 cmd/lambda/package/bootstrap
echo Binary MD5 $(cat cmd/lambda/package/bootstrap | md5sum )
## Why?
## https://zerostride.medium.com/building-deterministic-zip-files-with-built-in-commands-741275116a19
touch cmd/lambda/package/bootstrap -t 201301250000
(cd cmd/lambda/package; rm -f ../package.zip; zip -rq -D -X -9 -A --compression-method deflate ../package.zip bootstrap;)

echo Zip MD5 $(cat cmd/lambda/package.zip | md5sum )

$build -o cmd/auditLambda/package/bootstrap cmd/auditLambda/main.go
chmod 777 cmd/auditLambda/package/bootstrap
echo Binary MD5 $(cat cmd/auditLambda/package/bootstrap | md5sum )
touch cmd/auditLambda/package/bootstrap -t 201301250000
(cd cmd/auditLambda/package; rm -f ../package.zip; zip -rq -D -X -9 -A --compression-method deflate ../package.zip bootstrap;)

echo Zip MD5 $(cat cmd/auditLambda/package.zip | md5sum )