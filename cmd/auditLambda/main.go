package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	httplambda "github.com/metalblueberry/acnil-bot/pkg/httpLambda"
	"github.com/metalblueberry/acnil-bot/pkg/recipes"
	"github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"
	tele "gopkg.in/telebot.v3"
)

func Handler(b *tele.Bot) func(ctx context.Context, request httplambda.Request) error {
	return func(ctx context.Context, request httplambda.Request) error {
		log.Println("Handling request")

		update := telebot.Update{}
		err := json.Unmarshal([]byte(request.Body), &update)
		if err != nil {
			return err
		}

		log.Println("sending update, ", update.ID)
		b.ProcessUpdate(update)
		return nil
	}
}

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
