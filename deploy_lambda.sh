#!/bin/bash

set -e 
set -o pipefail

go test ./...

# GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o lambda/bootstrap cmd/lambda/main.go
# cp credentials.json lambda/credentials.json

# GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o auditLambda/bootstrap cmd/auditLambda/main.go
# cp credentials.json auditLambda/credentials.json

terraform -chdir=./tf apply -input=false \
     -var=sheets_private_key="$SHEETS_PRIVATE_KEY" \
     -var=sheets_private_key_id="$SHEETS_PRIVATE_KEY_ID" \
     -var=bot_token=$TOKEN \
     -var=sheet_id=$SHEET_ID \
     -var=audit_sheet_id=$AUDIT_SHEET_ID

export FUNCTION_URL=$(terraform -chdir=./tf output function_url)  

echo "Bot token selected"
curl -H "Content-Type: application/json"  -X GET "https://api.telegram.org/bot$TOKEN/getMe"
echo "Configure webhook"
curl -H "Content-Type: application/json"  -X POST "https://api.telegram.org/bot$TOKEN/setWebhook" -d "{
     \"url\": $FUNCTION_URL
     }"