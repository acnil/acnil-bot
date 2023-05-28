package acnil

import (
	"context"
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/metalblueberry/acnil-bot/pkg/sheetsparser"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"google.golang.org/api/sheets/v4"
)

type SheetGameDatabase struct {
	SRV       *sheets.Service
	ReadRange string
	Sheet     string
	SheetID   string
}

func NewGameDatabase(srv *sheets.Service, sheetID string) *SheetGameDatabase {
	return &SheetGameDatabase{
		SRV:       srv,
		ReadRange: "A:T",
		Sheet:     "Juegos de mesa",
		SheetID:   sheetID,
	}
}

const (
	NCols = 8
)

func (db *SheetGameDatabase) fullReadRange() string {
	return fmt.Sprintf("%s!%s", db.Sheet, db.ReadRange)
}

func (db *SheetGameDatabase) rowReadRange(row int) string {
	return fmt.Sprintf("%s!%d:%d", db.Sheet, row, row)
}

func (db *SheetGameDatabase) Get(ctx context.Context, id string, name string) (*Game, error) {
	games, err := db.List(ctx)
	if err != nil {
		return nil, err
	}

	return Games(games).Get(id, name)
}

func (db *SheetGameDatabase) List(ctx context.Context) ([]Game, error) {
	resp, err := db.SRV.Spreadsheets.Values.Get(db.SheetID, db.fullReadRange()).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	games := []Game{}

	if len(resp.Values) == 0 {
		return games, nil
	}

	for i, row := range resp.Values[1:] {
		if len(row) < NCols {
			continue
		}
		g := Game{
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

func (db *SheetGameDatabase) Find(ctx context.Context, name string) ([]Game, error) {
	games, err := db.List(ctx)
	if err != nil {
		return nil, err
	}

	matches := []Game{}

	for _, g := range games {
		if strings.Contains(
			Norm(g.Name),
			Norm(name),
		) {
			matches = append(matches, g)
		}
	}
	return matches, nil
}

func (db *SheetGameDatabase) Update(ctx context.Context, games ...Game) error {

	batchUpdate := &sheets.BatchUpdateValuesRequest{
		Data:             []*sheets.ValueRange{},
		ValueInputOption: "USER_ENTERED",
	}

	for _, game := range games {

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

//Norm normalises a string for comparison
func Norm(in string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	dst := make([]byte, len(in))
	ndst, _, err := t.Transform(dst, []byte(in), true)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return strings.ToLower(in)
	}
	return strings.ToLower(string(dst[:ndst]))
}
