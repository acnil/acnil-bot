package acnil

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/metalblueberry/acnil-bot/pkg/bgg"
	"github.com/metalblueberry/acnil-bot/pkg/sheetsparser"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"google.golang.org/api/sheets/v4"
	tele "gopkg.in/telebot.v3"
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
		ReadRange: "A:L",
		Sheet:     "Juegos de mesa",
		SheetID:   sheetID,
	}
}

type Game struct {
	// Row represents the row definition on google sheets
	Row string

	ID         string    `col:"0"`
	Name       string    `col:"1"`
	Location   string    `col:"2"`
	Holder     string    `col:"3"`
	Comments   string    `col:"4"`
	TakeDate   time.Time `col:"5"`
	ReturnDate time.Time `col:"6"`
	Price      string    `col:"7"`
	Publisher  string    `col:"8"`
	BGG        string    `col:"9"`
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

type MultipleMatchesError struct {
	Matches []Game
}

func (err MultipleMatchesError) Error() string {
	return "Wops! Parece que hay mas de un juego con este id y nombre, modifica el excel manualmente para asegurar que no hay nombres idénticos."
}

func (db *SheetGameDatabase) Get(ctx context.Context, id string, name string) (*Game, error) {
	games, err := db.List(ctx)
	if err != nil {
		return nil, err
	}

	matches := []Game{}

	for _, g := range games {
		if g.Matches(id, name) {
			matches = append(matches, g)
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}

	if len(matches) != 1 {
		return nil, MultipleMatchesError{
			Matches: matches,
		}
	}

	return &matches[0], nil
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

func (db *SheetGameDatabase) Update(ctx context.Context, game Game) error {
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

func (g Game) ToRow() (range_ string, row []interface{}) {
	v, err := sheetsparser.Marshal(&g)
	if err != nil {
		panic(err)
	}
	return g.Row, v
}

func (g Game) Matches(id string, name string) bool {
	return (Norm(g.Name) == Norm(name) || name == "") && (g.ID == id || id == "")
}

func (g Game) MatchesGame(game Game) bool {
	return g.Matches(game.ID, game.Name)
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

func (g Game) Buttons(member Member) *tele.ReplyMarkup {
	selector := &tele.ReplyMarkup{}
	data := g.Data()
	rows := []tele.Row{}
	switch {
	case g.Holder == "":
		rows = append(rows, selector.Row(
			selector.Data("Tomar Prestado", "take", data),
		))
	case member.Nickname == g.Holder:
		rows = append(rows, selector.Row(
			selector.Data("Devolver", "return", data),
		))
	}
	rows = append(rows, selector.Row(
		selector.Data("Mas información", "more", data),
	))

	selector.Inline(rows...)

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
{{ define "card" }}
{{if .ID }}ID: {{ .ID }}{{end}}
{{ .Name }}
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

{{ end }}

{{ define "morecard" }}
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
{{ end }}
`))
)

func (g Game) Card() string {
	b := &bytes.Buffer{}
	tmpl.ExecuteTemplate(b, "card", g)
	return b.String()
}

func (g Game) MoreCard() string {
	b := &bytes.Buffer{}
	tmpl.ExecuteTemplate(b, "morecard", g)
	return b.String()
}

func (g Game) Available() bool {
	return g.Holder == ""
}

func (g Game) String() string {
	if g.Available() {
		return fmt.Sprintf("🟢 %04s: %s", g.ID, g.Name)
	}
	return fmt.Sprintf("🔴 %04s: %s", g.ID, g.Name)
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

func (g Game) Equals(other Game) bool {
	// Row doesn't mater
	g.Row = ""
	other.Row = ""
	return reflect.DeepEqual(g, other)
}
