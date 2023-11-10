package acnil

import (
	"context"
	"fmt"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/sheetsparser"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/sheets/v4"
)

type JuegatronAuditDatabase interface {
	Append(ctx context.Context, entries []JuegatronAuditEntry) error
	List(ctx context.Context) ([]JuegatronAuditEntry, error)
	Delete(ctx context.Context, entry JuegatronAuditEntry) error
}

type JuegatronAuditEntry struct {
	// Row represents the row definition on google sheets
	Row       string
	ID        string `col:"0"`
	Holder    string `col:"2"`
	Actor     string `col:"3"`
	Timestamp string `col:"4"`
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
		Sheet:     "Préstamos",
		SheetID:   sheetID,

		parser: sheetsparser.SheetParser{
			DateFormat: time.RFC3339,
		},
	}
}

func (db *JuegatronSheetAuditDatabase) List(ctx context.Context) ([]JuegatronAuditEntry, error) {
	resp, err := db.SRV.Spreadsheets.Values.Get(db.SheetID, db.fullReadRange()).Do()
	if err != nil {
		logrus.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	games := []JuegatronAuditEntry{}

	if len(resp.Values) == 0 {
		return games, nil
	}

	for i, row := range resp.Values[1:] {
		if len(row) < NCols {
			continue
		}
		g := JuegatronAuditEntry{
			Row: db.rowReadRange(i + 2),
		}
		err := sheetsparser.Unmarshal(row, &g)
		if err != nil {
			return nil, err
		}
		games = append(games, g)

	}
	return games, nil
}

func (db *JuegatronSheetAuditDatabase) Delete(ctx context.Context, entry JuegatronAuditEntry) error {
	entry.Actor = ""
	entry.Holder = ""
	entry.ID = ""
	entry.Timestamp = ""
	return db.Update(ctx, entry)
}

func (db *JuegatronSheetAuditDatabase) Update(ctx context.Context, entries ...JuegatronAuditEntry) error {
	batchUpdate := &sheets.BatchUpdateValuesRequest{
		Data:             []*sheets.ValueRange{},
		ValueInputOption: "USER_ENTERED",
	}

	for _, game := range entries {

		rows := [][]interface{}{}
		row, err := sheetsparser.Marshal(&game)
		if err != nil {
			return fmt.Errorf("Failed to marshal game, %w", err)
		}
		rows = append(rows, row)
		batchUpdate.Data = append(batchUpdate.Data, &sheets.ValueRange{
			Range:  game.Row,
			Values: rows,
		})
	}

	request := db.SRV.Spreadsheets.Values.BatchUpdate(db.SheetID, batchUpdate)
	_, err := request.Do()
	if err != nil {
		return err
	}
	return nil
}

func (db *JuegatronSheetAuditDatabase) Append(ctx context.Context, entries []JuegatronAuditEntry) error {
	rows := [][]interface{}{}

	for _, entry := range entries {
		entry.Timestamp = time.Now().Format(db.parser.DateFormat)
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
