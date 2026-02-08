package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/acnil/acnil-bot/pkg/acnil"
	httplambda "github.com/acnil/acnil-bot/pkg/httpLambda"
	"github.com/acnil/acnil-bot/pkg/recipes"
	"github.com/aws/aws-lambda-go/lambda"
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
		httplambda.SetContext(update.ID, ctx)
		defer httplambda.DeleteContext(update.ID)

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
	juegatronSheetID := os.Getenv("JUEGATRON_SHEET_ID")
	if juegatronSheetID == "" {
		logrus.Fatal("JUEGATRON_SHEET_ID must be defined")
	}

	auditBackend := getEnv("AUDIT_BACKEND", "sheets")

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

	// Configure audit database based on backend selection
	var auditQuery acnil.ROAudit
	switch auditBackend {
	case "postgres":
		// PostgreSQL backend for audit only
		dbConfig := acnil.NewDatabaseConfigFromEnv()
		db, err := dbConfig.Connect()
		if err != nil {
			logrus.Fatalf("Failed to connect to audit database: %v", err)
		}
		defer db.Close()

		auditQuery = &acnil.AuditQuery{
			AuditDB: acnil.NewPostgresAuditDatabase(db),
		}
		logrus.Info("Using PostgreSQL backend for audit")

	case "sheets":
		// Google Sheets backend for audit (default)
		auditQuery = &acnil.AuditQuery{
			AuditDB: acnil.NewSheetAuditDatabase(srv, auditSheetID),
		}
		logrus.Info("Using Google Sheets backend for audit")

	default:
		logrus.Fatalf("Unknown audit backend: %s. Supported backends: sheets, postgres", auditBackend)
	}

	juegatronAudit := &acnil.JuegatronAudit{
		AuditDB: acnil.NewJuegatronSheetAuditDatabase(srv, juegatronSheetID),
	}

	// All other databases remain in Google Sheets
	handler := &acnil.Handler{
		MembersDB:       acnil.NewMembersDatabase(srv, sheetID),       // Google Sheets
		GameDB:          acnil.NewGameDatabase(srv, sheetID),          // Google Sheets
		JuegatronGameDB: acnil.NewGameDatabase(srv, juegatronSheetID), // Google Sheets
		JuegatronAudit:  juegatronAudit,                               // Google Sheets
		Audit:           auditQuery,                                   // PostgreSQL or Sheets
		Bot:             b,
	}

	handlerGroup := b.Group()
	handler.Register(handlerGroup)

	logrus.Println("starting lambda")
	lambda.Start(Handler(b))
}

func getEnv(key string, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}
