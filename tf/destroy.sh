#!/bin/bash
set -e 
set -o pipefail


terraform -chdir=./tf destroy -input=false -no-color -auto-approve \
     -var=sheets_private_key="$SHEETS_PRIVATE_KEY" \
     -var=sheets_private_key_id="$SHEETS_PRIVATE_KEY_ID" \
     -var=bot_token=$TOKEN \
     -var=sheet_id=$SHEET_ID \
     -var=audit_sheet_id=$AUDIT_SHEET_ID

curl -H "Content-Type: application/json"  -X POST "https://api.telegram.org/bot$TOKEN/setWebhook" -d "{
     \"url\": \"\" 
     }"
