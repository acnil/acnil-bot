package acnil

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/metalblueberry/acnil-bot/pkg/bgg"
	"google.golang.org/api/sheets/v4"
	tele "gopkg.in/telebot.v3"
)

type GameDatabase struct {
	SRV       *sheets.Service
	ReadRange string
	Sheet     string
	SheetID   string
}

type Game struct {
	ID        string
	Row       string
	Name      string
	Price     string
	Holder    string
	Location  string
	Publisher string
	Comments  string
}

func (db *GameDatabase) fullReadRange() string {
	return fmt.Sprintf("%s!%s", db.Sheet, db.ReadRange)
}
func (db *GameDatabase) rowReadRange(row int) string {
	return fmt.Sprintf("%s!%d:%d", db.Sheet, row, row)
}

func (db *GameDatabase) Get(ctx context.Context, name string) ([]Game, error) {
	games, err := db.List(ctx)
	if err != nil {
		return nil, err
	}

	matches := []Game{}

	for _, g := range games {
		if strings.ToLower(g.Name) == strings.ToLower(name) {
			matches = append(matches, g)
		}
	}
	return matches, nil
}

func (db *GameDatabase) List(ctx context.Context) ([]Game, error) {
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
		games = append(games, NewGameFromRow(db.rowReadRange(i+2), row))
	}
	return games, nil
}

func (db *GameDatabase) Find(ctx context.Context, name string) ([]Game, error) {
	games, err := db.List(ctx)
	if err != nil {
		return nil, err
	}

	matches := []Game{}

	for _, g := range games {
		if strings.Contains(
			strings.ToLower(g.Name),
			strings.ToLower(name),
		) {
			matches = append(matches, g)
		}
	}
	return matches, nil
}

func (db *GameDatabase) Update(ctx context.Context, game Game) error {
	range_, row := game.ToRow()

	request := db.SRV.Spreadsheets.Values.Update(db.SheetID, game.Row, &sheets.ValueRange{
		Range: range_,
		Values: [][]interface{}{
			row,
		},
	})
	request.ValueInputOption("USER_ENTERED")
	_, err := request.Do()
	if err != nil {
		return err
	}
	return nil
}

const (
	ColumnName       = 0
	ColumnPrice      = 1
	ColumnHolder     = 2
	ColumnLocation   = 3
	ColumnTakeDate   = 4
	ColumnReturnDate = 5
	ColumnPublisher  = 7
	ColumnQuarantine = 8
	ColumnComments   = 9
	NCols            = ColumnPublisher
	MaxCols          = ColumnComments + 1
)

func NewGameFromRow(range_ string, row []interface{}) Game {
	fullrow := make([]string, MaxCols)
	for i := range fullrow {
		fullrow[i] = ""
	}
	for i := range row {
		fullrow[i] = row[i].(string)
	}
	return Game{
		Row:       range_, // Exclude header and set index to 1 based
		Name:      fullrow[ColumnName],
		Price:     fullrow[ColumnPrice],
		Holder:    fullrow[ColumnHolder],
		Location:  fullrow[ColumnLocation],
		Publisher: fullrow[ColumnPublisher],
		Comments:  fullrow[ColumnComments],
	}
}

func (g Game) ToRow() (range_ string, row []interface{}) {
	return g.Row, []interface{}{
		g.Name,
		nil,
		g.Holder,
	}
}

func NewGameFromData(data string) Game {
	fields := strings.SplitN(data, "|", 3)
	return Game{
		ID:   fields[0],
		Row:  fields[1],
		Name: fields[2],
	}
}

func (g Game) Data() string {
	return strings.Join([]string{g.ID, g.Row, g.Name}, "|")
}

func (g Game) Buttons(c tele.Context) *tele.ReplyMarkup {
	selector := &tele.ReplyMarkup{}
	data := g.Data()
	switch {
	case g.Holder == "":
		selector.Inline(selector.Row(
			selector.Data("Tomar Prestado", "take", data),
		))
	case c.Sender().Username == g.Holder:
		selector.Inline(selector.Row(
			selector.Data("Devolver", "return", data),
		))
	}
	return selector
}

var (
	bggClient = bgg.NewClient()
	tmpl      = template.Must(template.New("game").Funcs(template.FuncMap{
		"bgg": func(name string) string {
			sr, err := bggClient.Search(context.Background(), name)
			if err != nil {
				log.Println("failed to search", err)
				return ""
			}
			st := sr.First()
			if st == nil {
				log.Println("not found")
				return ""
			}
			return bggClient.ResolveHref(st.Href)
		},
	}).Parse(`
Juego: {{ .Name }}
Editorial: {{ .Publisher}}
Precio: {{ .Price }}
Ubicación: {{ .Location }}
{{ if eq 0 (len .Holder) -}}
🟢 Disponible
{{- else -}}
🔴 Ocupado: {{ .Holder -}}
{{ end }}

{{ if .Comments }}
Notas: {{ .Comments }}
{{ end }}
{{ .Name | bgg }}

	`))
)

func (g Game) Card() string {
	b := &bytes.Buffer{}
	tmpl.Execute(b, g)
	return b.String()
}

func (g Game) String() string {
	return g.Name
}
