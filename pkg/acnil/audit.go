package acnil

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/ilog"
	"github.com/metalblueberry/acnil-bot/pkg/sheetsparser"
	"github.com/sirupsen/logrus"
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

	_, err := db.SRV.Spreadsheets.Values.Append(db.SheetID, db.fullReadRange(), &sheets.ValueRange{Values: rows}).ValueInputOption("RAW").Do()
	if err != nil {
		log.Fatalf("Unable to append data to sheet: %v", err)
	}
	return nil
}

func (db *SheetAuditDatabase) List(ctx context.Context) ([]AuditEntry, error) {
	resp, err := db.SRV.Spreadsheets.Values.Get(db.SheetID, db.fullReadRange()).Do()
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

type AuditDatabase interface {
	// Find(ctx context.Context, name string) ([]AuditEntry, error)
	List(ctx context.Context) ([]AuditEntry, error)
	// Get(ctx context.Context, id string, name string) (*AuditEntry, error)
	Append(ctx context.Context, entries []AuditEntry) error
}

type AuditEntryType string

const (
	AuditEntryTypeNew     AuditEntryType = "new"
	AuditEntryTypeRemoved AuditEntryType = "removed"
	AuditEntryTypeUpdate  AuditEntryType = "update"
)

type AuditEntry struct {
	Timestamp time.Time      `col:"0"`
	Type      AuditEntryType `col:"1"`

	ID         string    `col:"2"`
	Name       string    `col:"3"`
	Location   string    `col:"4"`
	Holder     string    `col:"5"`
	Comments   string    `col:"6"`
	TakeDate   time.Time `col:"7"`
	ReturnDate time.Time `col:"8"`
	Price      string    `col:"9"`
	Publisher  string    `col:"10"`
	BGG        string    `col:"11"`
}

type ROGameDatabase interface {
	List(ctx context.Context) ([]Game, error)
}

type Snapshot []*Game

type Audit struct {
	AuditDB  AuditDatabase
	GameDB   ROGameDatabase
	snapshot Snapshot
}

func (a *Audit) Run(ctx context.Context) {

	log.Print("Update audit entry")
	err := a.Do(ctx)
	if err != nil {
		log.Printf("Failed to update audit!! %s", err)
	}

	lastUpdate := time.Now()
	ticker := time.NewTicker(time.Minute * 30)

	go func() {
		log.Print("Wait for ticket to track audit")
		select {
		case <-ticker.C:
			if lastUpdate.Before(time.Now().Add(time.Hour * -24)) {
				log.Print("Update audit entry")
				err := a.Do(ctx)
				if err != nil {
					log.Printf("Failed to update audit!! %s", err)
				}
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}()
}

func (a *Audit) Do(ctx context.Context) error {
	log := logrus.WithField(ilog.FieldHandler, "Audit")

	if a.snapshot == nil {
		// rebuild snapshot from audit events
		a.snapshot = []*Game{}
		entries, err := a.AuditDB.List(ctx)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			a.snapshot.UpdateOrInsert(entry.Game())
		}
		log.Infof("Snapshot contains %d entries", len(a.snapshot))
	}

	// get new games
	games, err := a.GameDB.List(ctx)
	if err != nil {
		return fmt.Errorf("Failed to build snapshot, %w", err)
	}

	newEntries := a.snapshot.diff(log, games)

	log.WithField("len", len(newEntries)).Info("New audit entries")

	for _, entry := range newEntries {
		a.snapshot.UpdateOrInsert(entry.Game())
	}

	safe := a.snapshot.diff(log, games)
	if len(safe) != 0 {
		return fmt.Errorf("Failed to calculate diff, it reports changes after being applied twice")
	}

	err = a.AuditDB.Append(ctx, newEntries)
	if err != nil {
		return fmt.Errorf("Failed to post audit update, %w", err)
	}
	return nil
}

func (e AuditEntry) Game() *Game {
	return &Game{
		Row:        "",
		ID:         e.ID,
		Name:       e.Name,
		Location:   e.Location,
		Holder:     e.Holder,
		Comments:   e.Comments,
		TakeDate:   e.TakeDate,
		ReturnDate: e.ReturnDate,
		Price:      e.Price,
		Publisher:  e.Publisher,
		BGG:        e.BGG,
	}
}

func (s *Snapshot) UpdateOrInsert(game *Game) {
	for i, g := range *s {
		if g.Matches(game.ID, game.Name) {
			(*s)[i] = game
			return
		}
	}

	(*s) = append((*s), game)
}

func (s *Snapshot) Find(game Game) *Game {
	for _, g := range *s {
		if g.Matches(game.ID, game.Name) {
			return g
		}
	}
	return nil
}

func (s Snapshot) diff(log *logrus.Entry, games []Game) []AuditEntry {
	newEntries := []AuditEntry{}
	for _, game := range games {
		game.Row = ""
		foundGame := s.Find(game)
		if foundGame == nil {
			log.
				WithField("Game", game).
				Info("Found new game ")
			newEntries = append(newEntries, NewAuditEntry(game, AuditEntryTypeNew))
			continue
		}
		if !foundGame.Equals(game) {
			foundGameB, _ := json.Marshal(foundGame)
			gameB, _ := json.Marshal(game)
			log.WithField("Before", string(foundGameB)).
				WithField("After", string(gameB)).
				Info("Found game updated")
			newEntries = append(newEntries, NewAuditEntry(game, AuditEntryTypeUpdate))
		}
	}
	for _, snapshotGame := range s {
		found := false
		for _, game := range games {
			if snapshotGame.MatchesGame(game) {
				found = true
				break
			}
		}
		if !found {
			log.
				WithField("Game", snapshotGame).
				Info("Deleted game")
			newEntries = append(newEntries, NewAuditEntry(*snapshotGame, AuditEntryTypeRemoved))
		}
	}

	return newEntries
}

func NewAuditEntry(game Game, entryType AuditEntryType) AuditEntry {
	return AuditEntry{
		Type: entryType,

		ID:         game.ID,
		Name:       game.Name,
		Location:   game.Location,
		Holder:     game.Holder,
		Comments:   game.Comments,
		TakeDate:   game.TakeDate,
		ReturnDate: game.ReturnDate,
		Price:      game.Price,
		Publisher:  game.Publisher,
		BGG:        game.BGG,
	}
}
