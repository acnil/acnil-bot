package acnil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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
