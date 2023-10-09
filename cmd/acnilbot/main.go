package main

import (
	"context"
	"os"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	"github.com/metalblueberry/acnil-bot/pkg/recipes"
	"github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

func main() {

	disableAudit := os.Getenv("DISABLE_AUDIT")

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

	srv := recipes.SheetsService()

	if disableAudit == "" {
		audit := &acnil.Audit{
			AuditDB:   acnil.NewSheetAuditDatabase(srv, auditSheetID),
			GameDB:    acnil.NewGameDatabase(srv, sheetID),
			MembersDB: acnil.NewMembersDatabase(srv, sheetID),
			Bot:       b,
		}
		audit.Run(context.Background())
	}

	auditQuery := &acnil.AuditQuery{
		AuditDB: acnil.NewSheetAuditDatabase(srv, auditSheetID),
	}

	handler := &acnil.Handler{
		MembersDB: acnil.NewMembersDatabase(srv, sheetID),
		GameDB:    acnil.NewGameDatabase(srv, sheetID),
		Audit:     auditQuery,
		Bot:       b,
	}

	handlerGroup := b.Group()
	handler.Register(handlerGroup)

	logrus.Println("Application ready! listening for events")
	b.Start()
}

func GetEnv(key string, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}
