#!/bin/bash

set -e 
set -o pipefail

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -tags lambda.norpc -o cmd/lambda/package/bootstrap cmd/lambda/main.go
echo Binary MD5 $(cat cmd/lambda/package/bootstrap | md5sum )
## Why?
## https://zerostride.medium.com/building-deterministic-zip-files-with-built-in-commands-741275116a19
# chmod 777 cmd/lambda/package/bootstrap
touch cmd/lambda/package/bootstrap -t 201301250000
(cd cmd/lambda/package; rm -f ../package.zip; zip -rq -D -X -9 -A --compression-method deflate ../package.zip bootstrap;)

echo Zip MD5 $(cat cmd/lambda/package.zip | md5sum )

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -tags lambda.norpc -o cmd/auditLambda/package/bootstrap cmd/auditLambda/main.go
echo Binary MD5 $(cat cmd/auditLambda/package/bootstrap | md5sum )
# chmod 777 cmd/auditLambda/package/bootstrap
touch cmd/auditLambda/package/bootstrap -t 201301250000
(cd cmd/auditLambda/package; rm -f ../package.zip; zip -rq -D -X -9 -A --compression-method deflate ../package.zip bootstrap;)

echo Zip MD5 $(cat cmd/auditLambda/package.zip | md5sum )

terraform -chdir=./tf apply -input=false --auto-approve \
     -var=sheets_private_key="$SHEETS_PRIVATE_KEY" \
     -var=sheets_private_key_id="$SHEETS_PRIVATE_KEY_ID" \
     -var=sheets_email="$SHEETS_EMAIL" \
     -var=bot_token=$TOKEN \
     -var=sheet_id=$SHEET_ID \
     -var=audit_sheet_id=$AUDIT_SHEET_ID


echo "Bot token selected"
curl -H "Content-Type: application/json" -X GET "https://api.telegram.org/bot$TOKEN/getMe"

FUNCTION_URL=`terraform -chdir=./tf output function_url`
echo  ""
echo  "Configure webhook to $FUNCTION_URL"

echo curl -H "Content-Type: application/json" -X POST "https://api.telegram.org/bot$TOKEN/setWebhook" -d "{
     \"url\": $FUNCTION_URL
     }"

curl -H "Content-Type: application/json" -X POST "https://api.telegram.org/bot$TOKEN/setWebhook" -d "{
     \"url\": $FUNCTION_URL
     }"