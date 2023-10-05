#!/bin/bash

set -e 
set -o pipefail

export CGO_ENABLED=0 
export GOOS=linux 
export GOARCH=amd64
export GOPATH=/home/runner/go

flags="go build -trimpath -tags lambda.norpc -buildvcs=false -compiler gc"
go env

$flags -o cmd/lambda/package/bootstrap cmd/lambda/main.go

echo Binary MD5 $(cat cmd/lambda/package/bootstrap | md5sum )
## Why?
## https://zerostride.medium.com/building-deterministic-zip-files-with-built-in-commands-741275116a19
# chmod 777 cmd/lambda/package/bootstrap
touch cmd/lambda/package/bootstrap -t 201301250000
(cd cmd/lambda/package; rm -f ../package.zip; zip -rq -D -X -9 -A --compression-method deflate ../package.zip bootstrap;)

echo Zip MD5 $(cat cmd/lambda/package.zip | md5sum )

$flags -o cmd/auditLambda/package/bootstrap cmd/auditLambda/main.go
echo Binary MD5 $(cat cmd/auditLambda/package/bootstrap | md5sum )
# chmod 777 cmd/auditLambda/package/bootstrap
touch cmd/auditLambda/package/bootstrap -t 201301250000
(cd cmd/auditLambda/package; rm -f ../package.zip; zip -rq -D -X -9 -A --compression-method deflate ../package.zip bootstrap;)

echo Zip MD5 $(cat cmd/auditLambda/package.zip | md5sum )