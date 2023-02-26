package acnil

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/sheets/v4"
)

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

func CreateClientFromCredentials(ctx context.Context, file string) (*sheets.Service, error) {
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

	// not needed for now, Just an experiment
	// client.Transport = &gzipTransport{
	// 	RoundTripper: client.Transport,
	// 	// Enabled:      true,
	// }

	// http.DefaultTransport.(*http.Transport).DisableCompression = true
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
