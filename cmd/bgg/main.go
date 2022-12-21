package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/manifoldco/promptui"
	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	"github.com/metalblueberry/acnil-bot/pkg/bgg"
	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
)

type ExtendedData struct {
	GameID             string `json:"game_id,omitempty"`
	GameName           string `json:"game_name,omitempty"`
	BGGID              string `json:"bggid,omitempty"`
	MinPlayers         string `json:"min_players,omitempty"`
	MaxPlayer          string `json:"max_player,omitempty"`
	Age                string `json:"age,omitempty"`
	MinPlaytime        string `json:"min_playtime,omitempty"`
	MaxPlaytime        string `json:"max_playtime,omitempty"`
	Playingtime        string `json:"playingtime,omitempty"`
	Yearpublished      string `json:"yearpublished,omitempty"`
	LanguageDependence string `json:"language_dependence,omitempty"`
}

type ExtendedDataDB struct {
	Games        []ExtendedData
	IndexBGGID   map[string]int
	IndexAcnilID map[string]int
}

func (db *ExtendedDataDB) LoadFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return db.Load(f)
}

func (db *ExtendedDataDB) Load(r io.Reader) error {
	err := json.NewDecoder(r).Decode(&db.Games)
	if err != nil {
		return err
	}
	db.IndexBGGID = map[string]int{}
	db.IndexAcnilID = map[string]int{}
	for i, g := range db.Games {
		db.IndexBGGID[g.BGGID] = i
		db.IndexAcnilID[g.GameID+g.GameName] = len(db.Games) - 1
	}
	return nil
}

func (db *ExtendedDataDB) SaveFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return db.Save(f)
}

func (db *ExtendedDataDB) Save(w io.Writer) error {
	return json.NewEncoder(w).Encode(&db.Games)
}

func (db *ExtendedDataDB) Append(new ExtendedData) {
	if db.IndexBGGID == nil {
		db.IndexBGGID = map[string]int{}
		db.IndexAcnilID = map[string]int{}
	}
	db.Games = append(db.Games, new)
	db.IndexBGGID[new.BGGID] = len(db.Games) - 1
	db.IndexAcnilID[new.GameID+new.GameName] = len(db.Games) - 1
}

func (db *ExtendedDataDB) GetByBGGID(ID string) (ExtendedData, bool) {
	v, ok := db.IndexBGGID[ID]
	if !ok {
		return ExtendedData{}, false
	}

	return db.Games[v], true
}

func (db *ExtendedDataDB) GetByAcnilID(ID string, Name string) (ExtendedData, bool) {
	v, ok := db.IndexAcnilID[ID+Name]
	if !ok {
		return ExtendedData{}, false
	}

	return db.Games[v], true
}

func main() {

	credentialsFile := GetEnv("CREDENTIALS_FILE", "credentials.json")
	sheetID := os.Getenv("SHEET_ID")
	if sheetID == "" {
		logrus.Fatal("SHEET_ID must be defined")
	}

	srv, err := acnil.CreateClientFromCredentials(context.Background(), credentialsFile)
	if err != nil {
		logrus.Fatal("Couldn't load credentials", err)
	}

	GameDB := acnil.NewGameDatabase(srv, sheetID)
	bggapi := bgg.NewClient()
	extended := ExtendedDataDB{}
	err = extended.LoadFile("extended.db.json")
	if err != nil {
		logrus.Warn("Failed to load db %s", err.Error)
	}

	games, err := GameDB.List(context.Background())
	if err != nil {
		logrus.Fatalf("couldn't list games, %s", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		extended.SaveFile("extended.db.json")
		os.Exit(1)
	}()

	for _, game := range games {
		if _, ok := extended.GetByAcnilID(game.ID, game.Name); ok {
			logrus.Infof("Already found %s, continue", game.Name)
			continue
		}

		logrus.Infof("Not found %s", game.Name)

		retry := true
		var bggGame *bgg.Boardgame
		for retry {
			retry = false
			bggGame, err = FindGame(bggapi, game)
			if err != nil {
				if Confirm("Retry?? " + err.Error()) {
					retry = true
					continue
				}
			}

		}
		if bggGame == nil {
			if !Confirm("Continue?") {
				extended.SaveFile("extended.db.json")
				break
			}
			continue
		}
		extended.Append(ExtendedData{
			GameID:             game.ID,
			GameName:           game.Name,
			BGGID:              bggGame.Objectid,
			LanguageDependence: bggGame.Poll.ByName("language_dependence").SingleResult().Value,
			MinPlayers:         bggGame.Minplayers,
			MaxPlayer:          bggGame.Maxplayers,
			Age:                bggGame.Age,
			MinPlaytime:        bggGame.Minplayers,
			MaxPlaytime:        bggGame.Maxplaytime,
			Playingtime:        bggGame.Playingtime,
			Yearpublished:      bggGame.Yearpublished,
		})
	}

}

func GetEnv(key string, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}

func FindGame(bggapi *bgg.Client, game acnil.Game) (*bgg.Boardgame, error) {
	search, err := bggapi.Search(context.Background(), game.Name)
	if err != nil {

	}

	if len(search.Items) == 0 {
		p := promptui.Prompt{
			Label: "Unable to find match, you must fill this manually, Please, enter BGG ID",
		}
		result, err := p.Run()
		if err != nil {
			return nil, err
		}

		logrus.Infof("Manual input for %s", game.Name)
		resp, err := bggapi.Get(context.Background(), result)
		if err != nil {
			return nil, fmt.Errorf("Unable to use manual input %w", err)
		}
		for _, i := range resp.Boardgame {
			log.Println(i.Name)
		}

		if len(resp.Boardgame) != 1 {
			return nil, fmt.Errorf("Unable to use manual input, not games found")

		}
		browser.OpenURL(bggapi.ResolveHref("boardgame/" + resp.Boardgame[0].Objectid))

		if Confirm("Is this game the right one?") {
			return &resp.Boardgame[0], nil
		}

		return nil, fmt.Errorf("Discarded manually")

	}
	first := search.First()

	if len(search.Items) == 1 {
		logrus.Infof("Single match for %s", game.Name)
		resp, err := bggapi.Get(context.Background(), first.ID)
		if err != nil {
			return nil, err
		}

		for _, i := range resp.Boardgame {
			log.Println(i.Name)
		}
		return &resp.Boardgame[0], nil
	}

	logrus.Info(game.Name)
	browser.OpenURL(bggapi.ResolveHref(first.Href))
	if Confirm("Is this game the right one?") {
		resp, err := bggapi.Get(context.Background(), first.ID)
		if err != nil {
			return nil, err
		}

		for _, i := range resp.Boardgame {
			log.Println(i.Name)
		}
		return &resp.Boardgame[0], nil
	}

retry:

	s := promptui.Select{
		Label: "Games found by name " + game.Name,
		Items: ToLabels(search.Items),
	}
	resultIndex, _, err := s.Run()
	if err != nil {
		return nil, err
	}

	logrus.Infof("Selected %s", search.Items[resultIndex])

	browser.OpenURL(bggapi.ResolveHref(search.Items[resultIndex].Href))

	if !Confirm("Check the browser, Is this the actual game?") {
		goto retry
	}

	resp, err := bggapi.Get(context.Background(), search.Items[resultIndex].ID)
	if err != nil {
		return nil, err
	}

	for _, i := range resp.Boardgame {
		log.Println(i.Name)
	}
	return &resp.Boardgame[0], nil

}

type Labeler interface {
	Label() string
}

func ToLabels[T Labeler](items []T) []string {
	labels := []string{}
	for _, i := range items {
		labels = append(labels, i.Label())
	}
	return labels
}

func Confirm(promt string) bool {
	prompt := promptui.Prompt{
		Label:     promt,
		IsConfirm: true,
	}

	result, _ := prompt.Run()
	if result == "y" {
		return true
	}
	return false

}
