package main

import (
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	"github.com/metalblueberry/acnil-bot/pkg/recipes"
	"github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

func main() {
	botToken := os.Getenv("TOKEN")
	if botToken == "" {
		logrus.Fatal("TOKEN must be defined")
	}

	sheetID := os.Getenv("SHEET_ID")
	if sheetID == "" {
		logrus.Fatal("SHEET_ID must be defined")
	}

	auditSheetID := os.Getenv("AUDIT_SHEET_ID")
	if auditSheetID == "" {
		logrus.Fatal("AUDIT_SHEET_ID must be defined")
	}

	srv := recipes.SheetsService()

	pref := tele.Settings{
		Token:       botToken,
		Synchronous: true,
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		logrus.Fatal(err)
		return
	}

	audit := &acnil.Audit{
		AuditDB:   acnil.NewSheetAuditDatabase(srv, auditSheetID),
		GameDB:    acnil.NewGameDatabase(srv, sheetID),
		MembersDB: acnil.NewMembersDatabase(srv, sheetID),
		Bot:       b,
	}
	log.Println("starting lambda")
	lambda.Start(audit.Do)
}

func GetEnv(key string, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}
