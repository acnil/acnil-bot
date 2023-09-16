package recipes

import (
	"compress/gzip"
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/sheets/v4"
)

// SheetsService  returns a sheet service loading credentials from environment
// to work with acnil bot
func SheetsService() *sheets.Service {
	SheetsPrivateKeyID := GetEnv("SHEETS_PRIVATE_KEY_ID", "")
	if SheetsPrivateKeyID == "" {
		logrus.Fatal("SHEETS_PRIVATE_KEY_ID must be defined")
	}

	SheetsPrivateKey := GetEnv("SHEETS_PRIVATE_KEY", "")
	if SheetsPrivateKey == "" {
		logrus.Fatal("SHEETS_PRIVATE_KEY must be defined")
	}

	credentialsFile := DefaultCredentialFile
	credentialsFile.PrivateKey = strings.ReplaceAll(SheetsPrivateKey, "\\n", "\n")
	credentialsFile.PrivateKeyID = SheetsPrivateKeyID

	srv, err := CreateClientFromCredentials(context.Background(), credentialsFile)
	if err != nil {
		panic(err)
	}
	return srv
}

type CredentialsFile struct {
	Email string `json:"email"`
	// Type                    string   `json:"type"`
	// ProjectID               string   `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	// ClientEmail             string   `json:"client_email"`
	// ClientID                string   `json:"client_id"`
	// AuthURL                 string   `json:"auth_url"`
	TokenURL string `json:"token_url"`
	// AuthProviderX509CERTURL string   `json:"auth_provider_x509_cert_url"`
	// ClientX509CERTURL       string   `json:"client_x509_cert_url"`
	Scopes []string `json:"scopes"`
}

// DefaultCredentialFile is the google account details for acnilbot
// PrivateKeyID and PrivateKey will be provided from secrets
var DefaultCredentialFile = CredentialsFile{
	Email:    "***REMOVED***",
	TokenURL: "https://oauth2.googleapis.com/token",
	Scopes: []string{
		"https://www.googleapis.com/auth/spreadsheets",
	},
}

func CreateClientFromCredentials(ctx context.Context, credentials CredentialsFile) (*sheets.Service, error) {
	// Create a JWT configurations object for the Google service account
	conf := &jwt.Config{
		Email:        credentials.Email,
		PrivateKey:   []byte(credentials.PrivateKey),
		PrivateKeyID: credentials.PrivateKeyID,
		TokenURL:     credentials.TokenURL,
		Scopes:       credentials.Scopes,
	}

	client := conf.Client(ctx)

	// not needed for now, Just an experiment
	// client.Transport = &gzipTransport{
	// 	RoundTripper: client.Transport,
	// 	// Enabled:      true,
	// }

	// http.DefaultTransport.(*http.Transport).DisableCompression = true
	// return sheets.NewService(ctx, option.WithHTTPClient(client))
	return sheets.New(client)
}

// gzipTransport is a simple implementation to send gzip request, Just because I CAN
type gzipTransport struct {
	http.RoundTripper
	Enabled bool
}

func (db gzipTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Ask for gzip content
	if db.Enabled {
		r.Header.Add("Accept-Encoding", "gzip")
	}

	resp, err := db.RoundTripper.RoundTrip(r)
	if err != nil {
		return resp, err
	}

	// if it is not gzip, just continue
	if !strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") || !db.Enabled {
		log.Print("gzip not supported")
		return resp, err
	}

	// prepare a gzip reader
	resp.Body, err = gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Header.Del("Content-Encoding")
	resp.Header.Del("Content-Length")
	resp.ContentLength = -1
	resp.Uncompressed = true

	return resp, nil
}

func GetEnv(key string, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}
