package bgg

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Client struct {
	Address    string
	XMLAddress string
	Client     Doer
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

const (
	EndpointSearch = "search/boardgame"
)

func NewClient() *Client {
	return &Client{
		Address:    "https://boardgamegeek.com/",
		XMLAddress: "https://api.geekdo.com/xmlapi/",
		Client:     http.DefaultClient,
	}
}

type SearchResponse struct {
	Items []Item `json:"items"`
}

func (sr SearchResponse) First() *Item {
	if len(sr.Items) > 0 {
		return &sr.Items[0]
	}
	return nil
}

type Item struct {
	Objectid      string     `json:"objectid"`
	Subtype       Subtype    `json:"subtype"`
	Primaryname   string     `json:"primaryname"`
	Nameid        string     `json:"nameid"`
	Yearpublished int64      `json:"yearpublished"`
	Ordtitle      string     `json:"ordtitle"`
	RepImageid    int64      `json:"rep_imageid"`
	Objecttype    Objecttype `json:"objecttype"`
	Name          string     `json:"name"`
	Sortindex     string     `json:"sortindex"`
	Type          Type       `json:"type"`
	ID            string     `json:"id"`
	Href          string     `json:"href"`
}

func (i Item) Label() string {
	return fmt.Sprintf("%s (%d)", i.Name, i.Yearpublished)
}

type Objecttype string

const (
	Thing Objecttype = "thing"
)

type Subtype string

const (
	SubtypeBoardgame Subtype = "boardgame"
)

type Type string

const (
	TypeThings Type = "things"
)

func (c *Client) ResolveHref(ref string) string {
	return c.Address + ref
}

func (c *Client) ResolveGameHref(id string) string {
	return c.Address + EndpointGet + id
}

func (c *Client) ResolveXMLHref(ref string) string {
	return c.XMLAddress + ref
}

func (c *Client) Search(ctx context.Context, query string) (*SearchResponse, error) {
	u, err := url.Parse(c.Address + EndpointSearch)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Add("q", query)
	values.Add("showcount", "20")
	values.Add("nosession", "1")
	u.RawQuery = values.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to build the request, %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to do the request, %w", err)

	}
	defer resp.Body.Close()

	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	sr := &SearchResponse{}
	err = json.Unmarshal(d, sr)
	if err != nil {
		return nil, fmt.Errorf("Failed to Unmarshal, %w, %s", err, string(d))
	}
	return sr, nil
}

// curl 'https://boardgamegeek.com/?q=bra&nosession=1&showcount=20' \
//   -H 'authority: boardgamegeek.com' \
//   -H 'accept: application/json, text/plain, */*' \
//   -H 'accept-language: es-ES,es;q=0.9,en;q=0.8' \
//   -H 'referer: https://boardgamegeek.com/' \
//   -H 'sec-ch-ua: "Chromium";v="106", "Google Chrome";v="106", "Not;A=Brand";v="99"' \
//   -H 'sec-ch-ua-mobile: ?0' \
//   -H 'sec-ch-ua-platform: "Linux"' \
//   -H 'sec-fetch-dest: empty' \
//   -H 'sec-fetch-mode: cors' \
//   -H 'sec-fetch-site: same-origin' \
//   -H 'user-agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36' \
//   --compressed
