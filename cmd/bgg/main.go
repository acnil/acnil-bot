package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/manifoldco/promptui"
	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	"github.com/metalblueberry/acnil-bot/pkg/bgg"
	"github.com/metalblueberry/acnil-bot/pkg/recipes"
	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type ExtendedData struct {
	Name               string  `json:"name,omitempty"`
	BGGID              string  `json:"bggid,omitempty"`
	MinPlayers         int     `json:"min_players,omitempty"`
	MaxPlayers         int     `json:"max_player,omitempty"`
	Age                int     `json:"age,omitempty"`
	Playingtime        float64 `json:"playingtime,omitempty"`
	Yearpublished      int     `json:"yearpublished,omitempty"`
	LanguageDependence string  `json:"language_dependence,omitempty"`
	AvgRate            float64 `json:"avg_rate,omitempty"`
	AvgWeight          float64 `json:"avg_weight,omitempty"`
}

type ExtendedDataDB struct {
	Games      []ExtendedData
	IndexBGGID map[string]int
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
	for i, g := range db.Games {
		db.IndexBGGID[g.BGGID] = i
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
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	return enc.Encode(&db.Games)
}

func (db *ExtendedDataDB) Append(new ExtendedData) {
	if db.IndexBGGID == nil {
		db.IndexBGGID = map[string]int{}
	}
	db.Games = append(db.Games, new)
	db.IndexBGGID[new.BGGID] = len(db.Games) - 1
}

func (db *ExtendedDataDB) Update(update ExtendedData) bool {
	i, ok := db.IndexBGGID[update.BGGID]
	if !ok {
		return false
	}
	db.Games[i] = update
	return true
}

func (db *ExtendedDataDB) GetByBGGID(ID string) (ExtendedData, bool) {
	v, ok := db.IndexBGGID[ID]
	if !ok {
		return ExtendedData{}, false
	}

	return db.Games[v], true
}

func main() {

	sheetID := os.Getenv("SHEET_ID")
	if sheetID == "" {
		logrus.Fatal("SHEET_ID must be defined")
	}

	srv := recipes.SheetsService()

	GameDB := acnil.NewGameDatabase(srv, sheetID)
	bggapi := bgg.NewClient()
	extended := &ExtendedDataDB{}
	err := extended.LoadFile("extended.db.json")
	if err != nil {
		logrus.Warnf("Failed to load db %s", err.Error())
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		extended.SaveFile("extended.db.json")
		os.Exit(1)
	}()

	app := cli.App{
		Name: "acnil-bgg",
		Commands: []*cli.Command{
			{
				Name:  "manual",
				Usage: "manually go over the game list and see if the links are correct",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "refresh",
					},
				},
				Action: func(ctx *cli.Context) error {
					return Manual(ctx, GameDB, bggapi, extended)
				},
			},
			{
				Name:  "fill-inventory",
				Usage: "Based on the current extended database, it will go to the inventory and fill the column with the IDs",
				Action: func(ctx *cli.Context) error {
					return FillInventory(ctx, GameDB, bggapi, extended)
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}

}
func FillInventory(ctx *cli.Context, GameDB acnil.GameDatabase, bggapi *bgg.Client, extended *ExtendedDataDB) error {
	games, err := GameDB.List(ctx.Context)
	if err != nil {
		logrus.Fatalf("couldn't list games, %s", err)
	}

	for i := range games {
		ex, ok := extended.GetByBGGID(games[i].BGG)
		if !ok {
			logrus.Warn("Failed to find game in database")
			continue
		}
		logrus.Infof("Found info for game %s: %s", ex.Name, ex.BGGID)

		games[i].BGG = ex.BGGID
		games[i].LanguageDependence = ex.LanguageDependence
		games[i].MinPlayers = ex.MinPlayers
		games[i].MaxPlayers = ex.MaxPlayers
		games[i].Age = ex.Age
		games[i].Playingtime = ex.Playingtime
		games[i].Yearpublished = ex.Yearpublished
		games[i].AvgRate = ex.AvgRate
		games[i].AvgWeight = ex.AvgWeight
	}

	return GameDB.Update(ctx.Context, games...)
}

func ExtendGameData(game *acnil.Game, ex ExtendedData) {
	game.BGG = ex.BGGID
	game.LanguageDependence = ex.LanguageDependence
	game.MinPlayers = ex.MinPlayers
	game.MaxPlayers = ex.MaxPlayers
	game.Age = ex.Age
	game.Playingtime = ex.Playingtime
	game.Yearpublished = ex.Yearpublished
	game.AvgRate = ex.AvgRate
	game.AvgWeight = ex.AvgWeight

}

func Manual(ctx *cli.Context, GameDB acnil.GameDatabase, bggapi *bgg.Client, extended *ExtendedDataDB) error {
	defer extended.SaveFile("extended.db.json")

	games, err := GameDB.List(ctx.Context)
	if err != nil {
		logrus.Fatalf("couldn't list games, %s", err)
	}

	for _, game := range games {
		if _, ok := extended.GetByBGGID(game.BGG); ok && !ctx.Bool("refresh") {
			logrus.Infof("Already found %s, continue", game.Name)
			continue
		}

		if game.BGG == "-" {
			logrus.Infof("Skip game due to manual nil value, %s", game.Name)
			continue
		}

		if game.BGG != "" {
			_, ok := extended.GetByBGGID(game.BGG)
			if ok {
				logrus.Infof("Found game in extended data, Updating information, %s", game.Name)
			} else {
				logrus.Infof("ID found in game database but not in Extended data, Fetching updated information, %s", game.Name)
			}

			bggGames, err := bggapi.Get(ctx.Context, game.BGG)
			if err != nil {
				logrus.Errorf("Failed to fetch game by ID in database, %s %s", game.Name, game.BGG)
				continue
			}
			if len(bggGames.Boardgame) != 1 {
				logrus.Errorf("Failed to find game by ID in database, %s %s", game.Name, game.BGG)
				continue
			}

			bggGame := bggGames.Boardgame[0]

			if ok {
				ok := extended.Update(NewExtendedDataFromBGGGame(bggGame))
				if !ok {
					logrus.Errorf("Failed to update extended data for game, %s", game.Name)
				}
			} else {
				extended.Append(NewExtendedDataFromBGGGame(bggGame))
			}
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
			if Confirm("Set manual skip?") {
				game.BGG = "-"
				if err := GameDB.Update(ctx.Context, game); err != nil {
					logrus.WithError(err).Error("Failed to update game")
				}
			}
			if !Confirm("Continue?") {
				extended.SaveFile("extended.db.json")
				break
			}
			continue
		}

		ex := NewExtendedDataFromBGGGame(*bggGame)
		ExtendGameData(&game, ex)
		if _, found := extended.GetByBGGID(bggGame.Objectid); found {
			logrus.Infof("Game Extended data Already found %s", game.Name)
		} else {
			logrus.Infof("Appending data for %s", game.Name)
			extended.Append(ex)
		}

		if err := GameDB.Update(ctx.Context, game); err != nil {
			logrus.WithError(err).Error("Failed to update game")
		} else {
			logrus.Infof("Updated game %s in database with ID %s", ex.Name, ex.BGGID)
		}
		extended.SaveFile("extended.db.json")
	}
	return nil
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
		return nil, fmt.Errorf("Failed to search: %w", err)
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
			logrus.Println(i.Name)
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
			logrus.Println(i.Name)
		}
		return &resp.Boardgame[0], nil
	}

	logrus.Infof("%s by %s", game.Name, game.Publisher)
	browser.OpenURL(bggapi.ResolveHref(first.Href))
	if Confirm("Is this game the right one?") {
		resp, err := bggapi.Get(context.Background(), first.ID)
		if err != nil {
			return nil, err
		}

		for _, i := range resp.Boardgame {
			logrus.Println(i.Name)
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

	logrus.Infof("Selected %v", search.Items[resultIndex])

	browser.OpenURL(bggapi.ResolveHref(search.Items[resultIndex].Href))

	if !Confirm("Check the browser, Is this the actual game?") {
		goto retry
	}

	resp, err := bggapi.Get(context.Background(), search.Items[resultIndex].ID)
	if err != nil {
		return nil, err
	}

	for _, i := range resp.Boardgame {
		logrus.Println(i.Name)
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

func NewExtendedDataFromBGGGame(bggGame bgg.Boardgame) ExtendedData {

	MustAtoi := func(s string) int {
		v, _ := strconv.Atoi(s)
		return v
	}

	MustFloat := func(s string) float64 {
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}
	return ExtendedData{
		Name:               bggGame.Name.Principal().Text,
		BGGID:              bggGame.Objectid,
		LanguageDependence: bggGame.Poll.ByName("language_dependence").SingleResult().Value,
		MinPlayers:         MustAtoi(bggGame.Minplayers),
		MaxPlayers:         MustAtoi(bggGame.Maxplayers),
		Age:                MustAtoi(bggGame.Age),
		Playingtime:        MustFloat(bggGame.Playingtime),
		Yearpublished:      MustAtoi(bggGame.Yearpublished),
		AvgRate:            MustFloat(bggGame.Statistics.Ratings.Average),
		AvgWeight:          MustFloat(bggGame.Statistics.Ratings.Averageweight),
	}
}
