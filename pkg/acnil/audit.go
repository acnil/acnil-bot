package acnil

import (
	"context"
	"fmt"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/ilog"
	"github.com/sirupsen/logrus"
)

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
	log := logrus.WithField(ilog.FieldHandler, "Audit Run")

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
		a.rebuildSnapshot(ctx, log)
	}

	games, err := a.GameDB.List(ctx)
	if err != nil {
		return fmt.Errorf("Failed to list game database, %w", err)
	}

	newEntries := a.calculateEntries(games)

	log.WithField("len", len(newEntries)).Info("New audit entries")

	if !a.isAppliedSuccessfully(games) {
		return fmt.Errorf("Failed to calculate diff, it reports changes after being applied twice")
	}

	err = a.AuditDB.Append(ctx, newEntries)
	if err != nil {
		// Invalidate snapshot due to failure on update
		a.snapshot = nil
		return fmt.Errorf("Failed to post audit update, %w", err)
	}
	return nil
}

func (a *Audit) rebuildSnapshot(ctx context.Context, log *logrus.Entry) error {
	// rebuild snapshot from audit events
	a.snapshot = []*Game{}
	entries, err := a.AuditDB.List(ctx)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		a.snapshot.ApplyEntry(entry)
	}
	log.WithField("len", len(a.snapshot)).Info("Rebuilding snapshot from audit events")
	return nil
}

func (a *Audit) calculateEntries(games []Game) []AuditEntry {
	newEntries := a.snapshot.diff(games)

	for _, entry := range newEntries {
		a.snapshot.ApplyEntry(entry)
	}

	return newEntries
}

func (a *Audit) isAppliedSuccessfully(games []Game) bool {
	safe := a.snapshot.diff(games)
	if len(safe) != 0 {
		return false
	}
	return true
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

func (s *Snapshot) ApplyEntry(entry AuditEntry) {
	switch entry.Type {
	case AuditEntryTypeNew:
		(*s) = append((*s), entry.Game())

	case AuditEntryTypeUpdate:
		game := entry.Game()
		for i, g := range *s {
			if g.Matches(game.ID, game.Name) {
				(*s)[i] = game
				return
			}
		}
	case AuditEntryTypeRemoved:
		game := entry.Game()
		for i, g := range *s {
			if g.Matches(game.ID, game.Name) {
				(*s)[i] = (*s)[len(*s)-1]
				(*s) = (*s)[:len(*s)-1]
				return
			}
		}
	}
}

func (s *Snapshot) Find(game Game) *Game {
	for _, g := range *s {
		if g.Matches(game.ID, game.Name) {
			return g
		}
	}
	return nil
}

func (s Snapshot) diff(games []Game) []AuditEntry {

	newEntries := []AuditEntry{}
	for _, game := range games {
		game.Row = ""
		foundGame := s.Find(game)
		if foundGame == nil {
			newEntries = append(newEntries, NewAuditEntry(game, AuditEntryTypeNew))
			continue
		}
		if !foundGame.Equals(game) {
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
