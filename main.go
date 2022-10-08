package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

type secret struct {
	Key string `json:"key,omitempty"`
}

func withApiKeyFromSecrets(file string) option.ClientOption {
	f, err := os.Open("secret.json")
	if err != nil {
		log.Fatal("unable to load credentials, %s", err)
	}
	defer f.Close()
	s := &secret{}
	err = json.NewDecoder(f).Decode(s)
	if err != nil {
		log.Fatalf("fail to decode secrets.json, %s", err)
	}
	return option.WithAPIKey(s.Key)
}

const (
	Sheet = "1WFJL0tNJ21XtHfy6ksJvRe5rSUeTG44BmQbeEWYo4Y0"
)

func main() {
	ctx := context.Background()

	srv, err := sheets.NewService(ctx, option.WithAPIKey("AIzaSyA69YWKm1VNyU-rI3Dh-pUTxvvCxTRK6jU"))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
	// ctx := context.Background()
	// b, err := ioutil.ReadFile("credentials.json")
	// if err != nil {
	// 	log.Fatalf("Unable to read client secret file: %v", err)
	// }

	// // If modifying these scopes, delete your previously saved token.json.
	// config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	// if err != nil {
	// 	log.Fatalf("Unable to parse client secret file to config: %v", err)
	// }
	// client := getClient(config)

	// srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	// if err != nil {
	// 	log.Fatalf("Unable to retrieve Sheets client: %v", err)
	// }

	// Prints the names and majors of students in a sample spreadsheet:
	readRange := "Historial de prestamos!A:C"
	insert := &sheets.ValueRange{}
	insert.Values = append(insert.Values, []interface{}{"hello", "testing", "insert"})

	request := srv.Spreadsheets.Values.Append(Sheet, readRange, insert)
	request.ValueInputOption("USER_ENTERED")

	resp, err := request.Do()
	if err != nil {
		panic(err)
	}
	log.Println(resp.MarshalJSON())

	// readRange = "Juegos de mesa!A:H"
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
	// 		fmt.Printf("%s\n", row)
	// 	}
	// }

}

type SheetDatabse struct {
}

type Table interface {
	Get(ctx context.Context, range_ string)
}

type Library struct {
}
