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

	backend := getEnv("BACKEND", "sheets")

	pref := tele.Settings{
		Token:       botToken,
		Synchronous: true,
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		logrus.Fatal(err)
		return
	}

	var audit *acnil.Audit

	switch backend {
	case "postgres":
		// PostgreSQL backend
		dbConfig := acnil.NewDatabaseConfigFromEnv()
		db, err := dbConfig.Connect()
		if err != nil {
			logrus.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		audit = &acnil.Audit{
			AuditDB:   acnil.NewPostgresAuditDatabase(db),
			GameDB:    acnil.NewPostgresGameDatabase(db),
			MembersDB: acnil.NewPostgresMembersDatabase(db),
			Bot:       b,
		}
		logrus.Info("Using PostgreSQL backend")

	case "sheets":
		// Google Sheets backend (default)
		sheetID := os.Getenv("SHEET_ID")
		if sheetID == "" {
			logrus.Fatal("SHEET_ID must be defined for sheets backend")
		}

		auditSheetID := os.Getenv("AUDIT_SHEET_ID")
		if auditSheetID == "" {
			logrus.Fatal("AUDIT_SHEET_ID must be defined for sheets backend")
		}

		srv := recipes.SheetsService()

		audit = &acnil.Audit{
			AuditDB:   acnil.NewSheetAuditDatabase(srv, auditSheetID),
			GameDB:    acnil.NewGameDatabase(srv, sheetID),
			MembersDB: acnil.NewMembersDatabase(srv, sheetID),
			Bot:       b,
		}
		logrus.Info("Using Google Sheets backend")

	default:
		logrus.Fatalf("Unknown backend: %s. Supported backends: sheets, postgres", backend)
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
