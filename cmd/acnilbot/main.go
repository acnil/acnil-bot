package main

import (
	"context"
	"os"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	"github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

func main() {

	credentialsFile := GetEnv("CREDENTIALS_FILE", "credentials.json")
	sheetID := os.Getenv("SHEET_ID")
	if sheetID == "" {
		logrus.Fatal("SHEET_ID must be defined")
	}

	botToken := os.Getenv("TOKEN")
	if botToken == "" {
		logrus.Fatal("TOKEN must be defined")
	}

	auditSheetID := os.Getenv("AUDIT_SHEET_ID")
	if auditSheetID == "" {
		logrus.Fatal("AUDIT_SHEET_ID must be defined")
	}

	pref := tele.Settings{
		Token:  botToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		logrus.Fatal(err)
		return
	}

	srv, err := acnil.CreateClientFromCredentials(context.TODO(), credentialsFile)
	if err != nil {
		panic(err)
	}
	handler := &acnil.Handler{
		MembersDB: acnil.NewMembersDatabase(srv, sheetID),
		GameDB:    acnil.NewGameDatabase(srv, sheetID),
	}

	handler.Register(b)

	audit := &acnil.Audit{
		AuditDB: acnil.NewSheetAuditDatabase(srv, auditSheetID),
		GameDB:  acnil.NewGameDatabase(srv, sheetID),
	}

	audit.Run(context.Background())

	b.Start()
}

func GetEnv(key string, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}
