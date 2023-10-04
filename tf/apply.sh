#!/bin/bash

set -e 
set -o pipefail

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -tags lambda.norpc -o cmd/lambda/package/bootstrap cmd/lambda/main.go
## Why?
## https://zerostride.medium.com/building-deterministic-zip-files-with-built-in-commands-741275116a19
touch cmd/lambda/package/bootstrap -t 201301250000
(cd cmd/lambda/package; rm ../package.zip; zip -rq -D -X -9 -A --compression-method deflate ../package.zip bootstrap;)

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -tags lambda.norpc -o cmd/auditLambda/package/bootstrap cmd/auditLambda/main.go
touch cmd/auditLambda/package/bootstrap -t 201301250000
(cd cmd/auditLambda/package; rm ../package.zip; zip -rq -D -X -9 -A --compression-method deflate ../package.zip bootstrap;)

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