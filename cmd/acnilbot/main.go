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

	juegatronSheetID := os.Getenv("JUEGATRON_SHEET_ID")
	if juegatronSheetID == "" {
		logrus.Fatal("JUEGATRON_SHEET_ID must be defined")
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

	juegatronAudit := &acnil.Audit{
		AuditDB:   acnil.NewSheetAuditDatabase(srv, juegatronSheetID),
		GameDB:    acnil.NewGameDatabase(srv, juegatronSheetID),
		MembersDB: acnil.NewMembersDatabase(srv, sheetID),
		Bot:       b,
	}

	if disableAudit == "" {
		audit := &acnil.Audit{
			AuditDB:   acnil.NewSheetAuditDatabase(srv, auditSheetID),
			GameDB:    acnil.NewGameDatabase(srv, sheetID),
			MembersDB: acnil.NewMembersDatabase(srv, sheetID),
			Bot:       b,
		}
		audit.Run(context.Background(), time.Hour)

		juegatronAudit.Run(context.Background(), time.Hour)
	}
	err = juegatronAudit.Do(context.Background())
	if err != nil {
		logrus.WithError(err).Error("Failed to update juegatron audit")
	}

	auditQuery := &acnil.AuditQuery{
		AuditDB: acnil.NewSheetAuditDatabase(srv, auditSheetID),
	}

	handler := &acnil.Handler{
		MembersDB:       acnil.NewMembersDatabase(srv, sheetID),
		GameDB:          acnil.NewGameDatabase(srv, sheetID),
		JuegatronGameDB: acnil.NewGameDatabase(srv, juegatronSheetID),
		JuegatronAudit:  juegatronAudit,
		Audit:           auditQuery,
		Bot:             b,
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
