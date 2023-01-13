package acnil

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/sheetsparser"
	"google.golang.org/api/sheets/v4"
)

type SheetAuditDatabase struct {
	SRV       *sheets.Service
	ReadRange string
	Sheet     string
	SheetID   string
	parser    sheetsparser.SheetParser
}

func NewSheetAuditDatabase(srv *sheets.Service, sheetID string) *SheetAuditDatabase {
	return &SheetAuditDatabase{
		SRV:       srv,
		ReadRange: "A:N",
		Sheet:     "Audit",
		SheetID:   sheetID,

		parser: sheetsparser.SheetParser{
			DateFormat: time.RFC3339,
		},
	}
}

func (db *SheetAuditDatabase) fullReadRange() string {
	return fmt.Sprintf("%s!%s", db.Sheet, db.ReadRange)
}
func (db *SheetAuditDatabase) rowReadRange(row int) string {
	return fmt.Sprintf("%s!%d:%d", db.Sheet, row, row)
}

func (db *SheetAuditDatabase) Append(ctx context.Context, entries []AuditEntry) error {
	rows := [][]interface{}{}

	for _, entry := range entries {
		entry.Timestamp = time.Now()
		row, err := db.parser.Marshal(&entry)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}

	_, err := db.SRV.Spreadsheets.Values.Append(db.SheetID, db.fullReadRange(), &sheets.ValueRange{Values: rows}).ValueInputOption("RAW").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("Unable to append data to sheet: %v", err)
	}
	return nil
}

func (db *SheetAuditDatabase) List(ctx context.Context) ([]AuditEntry, error) {
	resp, err := db.SRV.Spreadsheets.Values.Get(db.SheetID, db.fullReadRange()).Context(ctx).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	entries := []AuditEntry{}

	if len(resp.Values) == 0 {
		return entries, nil
	}

	for _, row := range resp.Values[1:] {
		if len(row) < NCols {
			continue
		}
		g := AuditEntry{}
		err := db.parser.Unmarshal(row, &g)
		if err != nil {
			return nil, err
		}
		entries = append(entries, g)

	}
	return entries, nil
}
