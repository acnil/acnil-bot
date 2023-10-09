package acnil

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/bgg"
	"github.com/metalblueberry/acnil-bot/pkg/sheetsparser"
	"github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

var (
	bggClient = bgg.NewClient()
	tmpl      = template.Must(template.New("game").Funcs(template.FuncMap{
		"bgg": func(game Game) string {
			if game.BGG != "" && game.BGG != "-" {
				return bggClient.ResolveGameHref(game.BGG)
			}
			return ""
		},
	}).Parse(`
{{ define "card" }}
{{if .ID }}ID: /{{ .ID }}{{end}}
{{ .Name }}
{{ .Location }}

{{ if .IsAvailable -}}
🟢 Disponible
{{- else -}}
🔴 Ocupado: {{ .Holder }} {{ .TakeDate.Format "2006-01-02" }} ({{ .LeaseDays }} días)
A devolver antes del {{ .ReturnDate.Format "2006-01-02" }} ({{ if .IsLeaseExpired }}⚠️ Ya deberías haber devuelto este juego{{ else }}{{ .ReturnInDays }} días{{ end -}})
{{ end }}

{{ if .Comments }}
Notas: 
{{ .Comments }}
{{ end }}

{{ end }}

{{ define "morecard" }}
{{if .ID }}ID: /{{ .ID }}{{end}}
{{ .Name }}
{{ .Publisher}} {{if .Price}}({{ .Price }}){{end}}
{{ .Location }}
{{- if .ContainsBGGData }}
Puntuación: {{ .AvgRate }}
Dificultad: {{ .AvgWeight }}
Edad: {{ .Age }}
Nº Jugadores: {{ .MinPlayers }}-{{.MaxPlayers}}
Tiempo de juego : {{ .Playingtime }}m
{{ if .LanguageDependence}}Dependencia del idioma:  {{ .LanguageDependence }} {{ end }}
{{ end }}
{{ if .IsAvailable -}}
🟢 Disponible
{{- else -}}
🔴 Ocupado: {{ .Holder }} 
Prestado el día {{ .TakeDate.Format "2006-01-02" }} ({{ .LeaseDays }} días)
A devolver antes del {{ .ReturnDate.Format "2006-01-02" }} ({{ if .IsLeaseExpired }}⚠️ Ya deberías haber devuelto este juego{{ else }}{{ .ReturnInDays }} días{{ end -}})
{{ end }}
{{- if .Comments }}
Notas: 
{{ .Comments }}
{{ end }}
{{ if .BGG -}} 
{{ bgg . }}
{{- end }}
{{ end }}
`))
)

type Game struct {
	// Row represents the row definition on google sheets
	Row string

	ID         string    `col:"0,ro"`
	Name       string    `col:"1,ro"`
	Location   string    `col:"2,ro"`
	Holder     string    `col:"3"`
	Comments   string    `col:"4"`
	TakeDate   time.Time `col:"5"`
	ReturnDate time.Time `col:"6,ro"`
	// ReturnDateFormula is the raw formula in the cell, this exists to allow reading the value but setting the formula
	ReturnDateFormula *string `col:"6,wo"`

	Price     string `col:"7"`
	Publisher string `col:"8"`
	BGG       string `col:"9"`

	AvgRate            float64 `col:"10"`
	AvgWeight          float64 `col:"11"`
	Age                int     `col:"12"`
	MinPlayers         int     `col:"13"`
	MaxPlayers         int     `col:"14"`
	Playingtime        float64 `col:"15"`
	Yearpublished      int     `col:"16"`
	LanguageDependence string  `col:"17"`
}

func NewGameFromData(data string) Game {
	fields := strings.SplitN(data, "|", 2)
	return Game{
		ID:   fields[0],
		Name: fields[1],
	}
}

var gameLineMatch = regexp.MustCompile(`[🔴🟢]\s[/]?0*(.*?):\s(.*?)(\s\((.*)\))?$`)

// NewGameFromLine Parses game information from a game line
// A game line is a line that contains structured information about a game
func NewGameFromLine(line string) (Game, error) {
	if !gameLineMatch.MatchString(line) {
		return Game{}, fmt.Errorf("%s Doesn't match the expression for a game line", line)
	}

	fragments := gameLineMatch.FindStringSubmatch(line)

	return Game{
		ID:     fragments[1],
		Name:   fragments[2],
		Holder: fragments[4],
	}, nil
}

func (g Game) String() string {
	if g.IsAvailable() {
		return fmt.Sprintf("🟢 /%04s: %s", g.ID, g.Name)
	}
	return fmt.Sprintf("🔴 /%04s: %s (%s)", g.ID, g.Name, g.Holder)
}

func (g Game) ContainsBGGData() bool {
	return g.BGG != "-" && g.BGG != ""
}

func (g *Game) SetLeaseTimeDays(days int) {
	if days < 0 {
		panic("lease time days must be greater than zero")
	}
	formula := fmt.Sprintf("=INDIRECT(ADDRESS(ROW();COLUMN()-1))+%d", days)
	g.ReturnDateFormula = &formula
	g.ReturnDate = g.TakeDate.Add(time.Hour * 24 * time.Duration(days))
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

// Because IDs may not be unique, this function will return true is both name and ID are the same.
// Including the case where ID is empty
func (g Game) IsTheSame(id string, name string) bool {
	return g.ID == id && Norm(g.Name) == Norm(name)
}

func (g Game) IsTheSameGame(game Game) bool {
	return g.IsTheSame(game.ID, game.Name)
}

func (g Game) IsHeldBy(member Member) bool {
	return strings.TrimSpace(Norm(g.Holder)) == Norm(member.Nickname)
}

func (g Game) LeaseDays() int {
	return int(g.LeaseDuration().Round(time.Hour*24).Hours()) / 24
}

func (g Game) LeaseDuration() time.Duration {
	return time.Now().Sub(g.TakeDate)
}

func (g Game) IsHeldForLongerThan(duration time.Duration) bool {
	return g.LeaseDuration() > duration
}

// IsLeaseExpired returns true if the game should have been returned today
// Be aware that if the date is before 2000, it will assume the date is wrong and return false
func (g Game) IsLeaseExpired() bool {
	// Ensure date is a real date, Excel displays 1900 when date is zero
	return g.ReturnDate.After(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) && g.ReturnDate.Before(time.Now())
}

// ReturnInDays gives the number of days until you have to return the game
// 0 means it is already expired
func (g Game) ReturnInDays() int {
	if g.IsLeaseExpired() {
		return 0
	}
	return int(g.ReturnDate.Sub(time.Now()).Hours()) / 24
}

func (g Game) Data() string {
	data := strings.Join([]string{g.ID, g.Name}, "|")
	return data
}

func (g Game) Buttons(member Member) *tele.ReplyMarkup {
	selector := &tele.ReplyMarkup{}
	data := g.Data()
	rows := []tele.Row{}

	if g.IsAvailable() {
		rows = append(rows, selector.Row(
			selector.Data("Tomar Prestado", "take", data),
		))
	} else {
		rows = append(rows, selector.Row(
			selector.Data("Devolver", "return", data),
		))
	}

	if g.ContainsBGGData() {
		rows = append(rows, selector.Row(
			selector.Data("Mas información", "more", data),
		))
	}

	rows = append(rows, selector.Row(
		selector.Data("Historial", "history", data),
	))

	if (g.IsHeldBy(member) || (member.Permissions == PermissionAdmin && !g.IsAvailable())) && g.IsLeaseExpired() {
		rows = append(rows, selector.Row(
			selector.Data("Dar mas tiempo", "extendLease", data),
		))
	}

	selector.Inline(rows...)

	return selector
}

func (g Game) Card() string {
	b := &bytes.Buffer{}
	err := tmpl.ExecuteTemplate(b, "card", g)
	if err != nil {
		logrus.Error("Unable to render template!!, ", err)
	}
	return b.String()
}

func (g Game) MoreCard() string {
	b := &bytes.Buffer{}
	err := tmpl.ExecuteTemplate(b, "morecard", g)
	if err != nil {
		logrus.Error("Unable to render template!!, ", err)
	}
	return b.String()
}

func (g Game) IsAvailable() bool {
	return g.Holder == ""
}

func (g Game) Equals(other Game) bool {
	// Row doesn't mater
	g.Row = ""
	other.Row = ""

	return g.ID == other.ID &&
		g.Name == other.Name &&
		g.Location == other.Location &&
		g.Holder == other.Holder &&
		g.Comments == other.Comments &&
		g.TakeDate == other.TakeDate &&
		g.ReturnDate == other.ReturnDate &&
		g.Price == other.Price &&
		g.Publisher == other.Publisher &&
		g.BGG == other.BGG
}

// Take sets the game holder to the given user and registers the take date
func (g *Game) Take(holder string) {
	g.Holder = holder
	g.TakeDate = time.Now()
	g.SetLeaseTimeDays(21)
}

// Return marks the game as returned
func (g *Game) Return() {
	g.Holder = ""
	g.TakeDate = time.Time{}
}

type Games []Game

type MultipleMatchesError struct {
	Matches []Game
}

func (err MultipleMatchesError) Error() string {
	return "Wops! Parece que hay mas de un juego con este id y nombre, modifica el excel manualmente para asegurar que no hay nombres idénticos."
}

func (games Games) Get(id string, name string) (*Game, error) {
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

func (games Games) Find(name string) []Game {

	matches := []Game{}

	for _, g := range games {
		if strings.Contains(
			Norm(g.Name),
			Norm(name),
		) {
			matches = append(matches, g)
		}
	}
	return matches
}

// CanReturn returns true if at least one game of the list can be returned
func (games Games) CanReturn() bool {
	for i := range games {
		if games[i].Holder != "" {
			return true
		}
	}
	return false
}

// CanTake returns true if at least one game of the list can be taken
func (games Games) CanTake() bool {
	for i := range games {
		if games[i].Holder == "" {
			return true
		}
	}
	return false
}

func (games Games) FindDuplicates() (duplicate Games, unique Games) {
	seen := map[string]bool{}
	for _, g := range games {
		if _, ok := seen[g.String()]; ok {
			duplicate = append(duplicate, g)
			continue
		}
		seen[g.String()] = true
		unique = append(unique, g)
	}
	return duplicate, unique
}
