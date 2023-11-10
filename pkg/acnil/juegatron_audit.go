package acnil

import (
	"context"
	"fmt"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/sheetsparser"
	"google.golang.org/api/sheets/v4"
)

type JuegatronAuditDatabase interface {
	Append(ctx context.Context, entries []JuegatronAuditEntry) error
}

type JuegatronAuditEntry struct {
	ID        string    `col:"0"`
	Holder    string    `col:"2"`
	Actor     string    `col:"3"`
	Timestamp time.Time `col:"4"`
}

type JuegatronAudit struct {
	AuditDB JuegatronAuditDatabase
}

func (e JuegatronAuditEntry) Game() *Game {
	return &Game{
		Row:    "",
		ID:     e.ID,
		Holder: e.Holder,
	}
}

func NewJuegatronAuditEntry(game Game, actor Member) JuegatronAuditEntry {
	entry := JuegatronAuditEntry{
		ID:     game.ID,
		Holder: game.Holder,
		Actor:  actor.Nickname,
	}
	if entry.Holder == "" {
		entry.Holder = "devuelto"
	}

	return entry
}

type JuegatronSheetAuditDatabase struct {
	SRV       *sheets.Service
	ReadRange string
	Sheet     string
	SheetID   string
	parser    sheetsparser.SheetParser
}

func NewJuegatronSheetAuditDatabase(srv *sheets.Service, sheetID string) *JuegatronSheetAuditDatabase {
	return &JuegatronSheetAuditDatabase{
		SRV:       srv,
		ReadRange: "A:N",
		Sheet:     "Audit",
		SheetID:   sheetID,

		parser: sheetsparser.SheetParser{
			DateFormat: time.RFC3339,
		},
	}
}

func (db *JuegatronSheetAuditDatabase) Append(ctx context.Context, entries []JuegatronAuditEntry) error {
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

func (db *JuegatronSheetAuditDatabase) fullReadRange() string {
	return fmt.Sprintf("%s!%s", db.Sheet, db.ReadRange)
}
func (db *JuegatronSheetAuditDatabase) rowReadRange(row int) string {
	return fmt.Sprintf("%s!%d:%d", db.Sheet, row, row)
}
