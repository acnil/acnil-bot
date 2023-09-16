#!/bin/bash

set -e 
set -o pipefail

GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o cmd/lambda/package/bootstrap cmd/lambda/main.go

GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o cmd/auditLambda/package/bootstrap cmd/auditLambda/main.go

terraform -chdir=./tf apply -input=false -no-color \
     -var=sheets_private_key="$SHEETS_PRIVATE_KEY" \
     -var=sheets_private_key_id="$SHEETS_PRIVATE_KEY_ID" \
     -var=bot_token=$TOKEN \
     -var=sheet_id=$SHEET_ID \
     -var=audit_sheet_id=$AUDIT_SHEET_ID


echo "Bot token selected"
curl -H "Content-Type: application/json"  -X GET "https://api.telegram.org/bot$TOKEN/getMe"

export FUNCTION_URL=$(terraform -chdir=./tf output function_url)  
echo "Configure webhook"
curl -H "Content-Type: application/json"  -X POST "https://api.telegram.org/bot$TOKEN/setWebhook" -d "{
     \"url\": $FUNCTION_URL
     }"