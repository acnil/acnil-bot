package acnil

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"text/template"
	"unicode"

	"github.com/metalblueberry/acnil-bot/pkg/bgg"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
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
	BGG       string
}

func (db *GameDatabase) fullReadRange() string {
	return fmt.Sprintf("%s!%s", db.Sheet, db.ReadRange)
}
func (db *GameDatabase) rowReadRange(row int) string {
	return fmt.Sprintf("%s!%d:%d", db.Sheet, row, row)
}

type MultipleMatchesError struct {
	Matches []Game
}

func (err MultipleMatchesError) Error() string {
	return "Wops! Parece que hay mas de un juego con este nombre, modifica el excel manualmente para asegurar que no hay nombres identicos."
}

func (db *GameDatabase) Get(ctx context.Context, id string, name string) (*Game, error) {
	games, err := db.List(ctx)
	if err != nil {
		return nil, err
	}

	matches := []Game{}

	for _, g := range games {
		if (Norm(g.Name) == Norm(name) || name == "") && (g.ID == id || id == "") {
			matches = append(matches, g)
		}
	}
	if len(matches) == 0 {
		return nil, err
	}

	if len(matches) != 1 {
		return nil, MultipleMatchesError{
			Matches: matches,
		}
	}

	return &matches[0], nil
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
			Norm(g.Name),
			Norm(name),
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
	ColumnID         = 0
	ColumnName       = 1
	ColumnPrice      = 2
	ColumnHolder     = 3
	ColumnLocation   = 4
	ColumnTakeDate   = 5
	ColumnReturnDate = 6
	ColumnPublisher  = 8
	ColumnQuarantine = 9
	ColumnComments   = 10
	ColumnBGG        = 11
	NCols            = ColumnPublisher
	MaxCols          = ColumnBGG + 1
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
		ID:        fullrow[ColumnID],
		Name:      fullrow[ColumnName],
		Price:     fullrow[ColumnPrice],
		Holder:    fullrow[ColumnHolder],
		Location:  fullrow[ColumnLocation],
		Publisher: fullrow[ColumnPublisher],
		Comments:  fullrow[ColumnComments],
		BGG:       fullrow[ColumnBGG],
	}
}

func (g Game) ToRow() (range_ string, row []interface{}) {
	return g.Row, []interface{}{
		g.ID,
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
{{if .ID }}ID: {{ .ID }}{{end}}
{{ .Name }}
{{ .Publisher}} ({{ .Price }})
{{ .Location }}

{{ if .Available -}}
🟢 Disponible
{{- else -}}
🔴 Ocupado: {{ .Holder -}}
{{ end }}

{{ if .Comments }}
Notas: 
{{ .Comments }}
{{ end }}
{{ if .BGG }} 
{{ .BGG}}
{{ else }}
{{ .Name | bgg }}
{{ end }}
`))
)

func (g Game) Card() string {
	b := &bytes.Buffer{}
	tmpl.Execute(b, g)
	return b.String()
}

func (g Game) Available() bool {
	return g.Holder == ""
}

func (g Game) String() string {
	if g.Available() {
		return "🟢 " + g.ID + ":" + g.Name
	}
	return "🔴 " + g.ID + ":" + g.Name
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
