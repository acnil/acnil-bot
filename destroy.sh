#!/bin/bash
set -e 
set -o pipefail


terraform -chdir=./tf destroy 

curl -H "Content-Type: application/json"  -X POST "https://api.telegram.org/bot$TOKEN/setWebhook" -d "{
     \"url\": \"\" 
     }"
