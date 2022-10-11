#!/bin/bash

env CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -o acnilbot cmd/acnilbot/main.go
ssh pi@192.168.1.139 sudo systemctl stop acnilbottest.service
scp ./acnilbot pi@192.168.1.139:/home/pi/acnilbottest/acnilbottest
ssh pi@192.168.1.139 sudo systemctl start acnilbottest.service
ssh pi@192.168.1.139 sudo systemctl status acnilbottest.service