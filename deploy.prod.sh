#!/bin/bash

env CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -o acnilbot cmd/acnilbot/main.go

ssh pi@192.168.1.139 sudo systemctl stop acnilbot.service

scp ./deploy/acnilbot.service pi@192.168.1.139:/home/pi/acnilbot/
ssh pi@192.168.1.139 sudo mv /home/pi/acnilbot/acnilbot.service /etc/systemd/system/acnilbot.service 
ssh pi@192.168.1.139 sudo systemctl daemon-reload 
scp ./acnilbot pi@192.168.1.139:/home/pi/acnilbot/acnilbot

ssh pi@192.168.1.139 sudo systemctl start acnilbot.service
ssh pi@192.168.1.139 sudo systemctl status acnilbot.service