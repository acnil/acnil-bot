package main

import (
	"os"

	"github.com/acnil/acnil-bot/pkg/acnil"
	"github.com/acnil/acnil-bot/pkg/recipes"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

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

	auditBackend := getEnv("AUDIT_BACKEND", "sheets")

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
	var auditDB acnil.AuditDatabase
	switch auditBackend {
	case "postgres":
		// PostgreSQL backend for audit only
		dbConfig := acnil.NewDatabaseConfigFromEnv()
		db, err := dbConfig.Connect()
		if err != nil {
			logrus.Fatalf("Failed to connect to audit database: %v", err)
		}
		defer db.Close()

		auditDB = acnil.NewPostgresAuditDatabase(db)
		logrus.Info("Using PostgreSQL backend for audit")

	case "sheets":
		// Google Sheets backend for audit (default)
		srv := recipes.SheetsService()
		auditDB = acnil.NewSheetAuditDatabase(srv, auditSheetID)
		logrus.Info("Using Google Sheets backend for audit")

	default:
		logrus.Fatalf("Unknown audit backend: %s. Supported backends: sheets, postgres", auditBackend)
	}

	// Game and member data always comes from Google Sheets (read-only for audit)
	srv := recipes.SheetsService()

	audit := &acnil.Audit{
		AuditDB:   auditDB,                                // PostgreSQL or Sheets
		GameDB:    acnil.NewGameDatabase(srv, sheetID),    // Google Sheets (read-only)
		MembersDB: acnil.NewMembersDatabase(srv, sheetID), // Google Sheets (for notifications)
		Bot:       b,
	}

	logrus.Println("starting lambda")
	lambda.Start(audit.Do)
}

func getEnv(key string, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}
