package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/sheets/v4"
)

const (
	SheetID = "1WFJL0tNJ21XtHfy6ksJvRe5rSUeTG44BmQbeEWYo4Y0"
)

// Generated by https://quicktype.io

type CredentialsFile struct {
	Email                   string   `json:"email"`
	Type                    string   `json:"type"`
	ProjectID               string   `json:"project_id"`
	PrivateKeyID            string   `json:"private_key_id"`
	PrivateKey              string   `json:"private_key"`
	ClientEmail             string   `json:"client_email"`
	ClientID                string   `json:"client_id"`
	AuthURL                 string   `json:"auth_url"`
	TokenURL                string   `json:"token_url"`
	AuthProviderX509CERTURL string   `json:"auth_provider_x509_cert_url"`
	ClientX509CERTURL       string   `json:"client_x509_cert_url"`
	Scopes                  []string `json:"scopes"`
}

func main() {

	// Prints the names and majors of students in a sample spreadsheet:
	// readRange := "Historial de prestamos!A:C"
	// insert := &sheets.ValueRange{}
	// insert.Values = append(insert.Values, []interface{}{"hello", "testing", "insert"})
	srv, err := CreateClientFromCredentals(context.Background(), "credentials.json")
	if err != nil {
		panic(err)
	}

	// srv.Spreadsheets.Values.Update(Sheet, readRange, &sheets.ValueRange{})

	// request := srv.Spreadsheets.Values.Append(Sheet, readRange, insert)
	// request.ValueInputOption("USER_ENTERED")

	// resp, err := request.Do()
	// if err != nil {
	// 	panic(err)
	// }
	// log.Println(resp.MarshalJSON())

	ctx := context.Background()

	db := GameDatabase{
		srv:       srv,
		readRange: "A:H",
		sheet:     "Juegos de mesa",
		sheetID:   SheetID,
	}
	list, err := db.Get(ctx)
	if err != nil {
		panic(err)
	}

	log.Println(list)

	g1 := list[0]

	g1.Holder = os.Args[1]
	log.Println(g1)
	err = db.Update(ctx, g1)
	if err != nil {
		panic(err)
	}

	// resp, err := srv.Spreadsheets.Values.Get(Sheet, readRange).Do()
	// if err != nil {
	// 	log.Fatalf("Unable to retrieve data from sheet: %v", err)
	// }

	// if len(resp.Values) == 0 {
	// 	fmt.Println("No data found.")
	// } else {
	// 	fmt.Println("Game:")
	// 	for _, row := range resp.Values {
	// 		// Print columns A and E, which correspond to indices 0 and 4.
	// 		v := []string{}
	// 		for _, el := range row {
	// 			v = append(v, el.(string))
	// 		}
	// 		fmt.Println(strings.Join(v, ","))
	// 	}
	// }

}

type GameDatabase struct {
	srv         *sheets.Service
	readRange   string
	sheet       string
	sheetID     string
	headerIndex map[string]int
}

func (db *GameDatabase) fullReadRange() string {
	return fmt.Sprintf("%s!%s", db.sheet, db.readRange)
}
func (db *GameDatabase) rowReadRange(row int) string {
	return fmt.Sprintf("%s!%d:%d", db.sheet, row, row)
}

func (db *GameDatabase) Get(ctx context.Context) ([]Game, error) {
	resp, err := db.srv.Spreadsheets.Values.Get(SheetID, db.fullReadRange()).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	games := []Game{}

	if len(resp.Values) == 0 {
		return games, nil
	}

	headerIndex := map[string]int{}
	for i, h := range resp.Values[0] {
		headerIndex[h.(string)] = i
	}
	db.headerIndex = headerIndex

	log.Println(headerIndex)

	for i, row := range resp.Values[1:] {
		log.Println(row)
		if len(row) < 7 {
			continue
		}
		games = append(games, Game{
			Row:    db.rowReadRange(i + 2), // Exclude header and set index to 1 based
			Name:   row[headerIndex["Nombre"]].(string),
			Holder: row[headerIndex["Prestado a"]].(string),
		})
	}
	return games, nil
}

func (db *GameDatabase) Update(ctx context.Context, game Game) error {
	row := make([]interface{}, len(db.headerIndex))
	row[0] = game.Name
	row[2] = game.Holder

	request := db.srv.Spreadsheets.Values.Update(SheetID, game.Row, &sheets.ValueRange{
		Range: game.Row,
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

type Library struct {
	GameTable *GameDatabase
}

type Game struct {
	ID        int
	Row       string
	Name      string
	Price     string
	Holder    string
	Premises  string
	Publisher string
	Notes     string
}

func (l *Library) FindByName(ctx context.Context, name string) {
	// l.GameTable.Get()
}

func CreateClientFromCredentals(ctx context.Context, file string) (*sheets.Service, error) {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	cf := &CredentialsFile{}

	err = json.NewDecoder(f).Decode(cf)
	if err != nil {
		return nil, fmt.Errorf("Cannot read %s file, %w", file, err)
	}

	// Create a JWT configurations object for the Google service account
	conf := &jwt.Config{
		Email:        cf.Email,
		PrivateKey:   []byte(cf.PrivateKey),
		PrivateKeyID: cf.PrivateKeyID,
		TokenURL:     cf.TokenURL,
		Scopes:       cf.Scopes,
	}

	client := conf.Client(ctx)
	return sheets.New(client)
}
