package main

import (
	"context"
	"encoding/json"
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
	webhookSecretToken := os.Getenv("WEBHOOK_SECRET_TOKEN")
	if webhookSecretToken == "" {
		logrus.Fatal("WEBHOOK_SECRET_TOKEN must be defined")
	}

	return func(ctx context.Context, request httplambda.Request) error {
		logrus.Println("Handling request")
		if request.Headers["x-telegram-bot-api-secret-token"] != webhookSecretToken {
			logrus.Printf("Request rejected because the token doesn't match, received %s", request.Headers)
			return nil
		}

		update := telebot.Update{}
		err := json.Unmarshal([]byte(request.Body), &update)
		if err != nil {
			return err
		}

		logrus.Println("sending update, ", update.ID)
		b.ProcessUpdate(update)
		return nil
	}
}

func main() {

	logrus.SetFormatter(&logrus.JSONFormatter{})

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

	logrus.Println("starting lambda")
	lambda.Start(Handler(b))
}

func GetEnv(key string, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}
