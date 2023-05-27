package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	"github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

func main() {

	disableAudit := os.Getenv("DISABLE_AUDIT")

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

	srv, err := acnil.CreateClientFromCredentials(context.Background(), credentialsFile)
	if err != nil {
		panic(err)
	}

	audit := &acnil.Audit{
		AuditDB:   acnil.NewSheetAuditDatabase(srv, auditSheetID),
		GameDB:    acnil.NewGameDatabase(srv, sheetID),
		MembersDB: acnil.NewMembersDatabase(srv, sheetID),
		Bot:       b,
	}

	if disableAudit == "" {
		audit.Run(context.Background())
	}

	handler := &acnil.Handler{
		MembersDB: acnil.NewMembersDatabase(srv, sheetID),
		GameDB:    acnil.NewGameDatabase(srv, sheetID),
		Audit:     audit,
	}

	handler.Register(b)

	log.Println("Application ready! listening for events")
	b.Start()
}

func GetEnv(key string, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}
