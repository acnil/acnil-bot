#!/bin/bash

set -e 
set -o pipefail

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o cmd/lambda/package/bootstrap cmd/lambda/main.go

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o cmd/auditLambda/package/bootstrap cmd/auditLambda/main.go

terraform -chdir=./tf plan -input=false \
     -var=sheets_private_key="$SHEETS_PRIVATE_KEY" \
     -var=sheets_private_key_id="$SHEETS_PRIVATE_KEY_ID" \
     -var=sheets_email="$SHEETS_EMAIL" \
     -var=bot_token=$TOKEN \
     -var=sheet_id=$SHEET_ID \
     -var=audit_sheet_id=$AUDIT_SHEET_ID

echo "Bot token selected"
curl -H "Content-Type: application/json"  -X GET "https://api.telegram.org/bot$TOKEN/getMe"