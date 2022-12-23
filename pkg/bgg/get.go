package bgg

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

type Boardgames struct {
	XMLName    xml.Name    `xml:"boardgames"`
	Text       string      `xml:",chardata"`
	Termsofuse string      `xml:"termsofuse,attr"`
	Boardgame  []Boardgame `xml:"boardgame"`
}

type Boardgame struct {
	Text          string `xml:",chardata"`
	Objectid      string `xml:"objectid,attr"`
	Yearpublished string `xml:"yearpublished"`
	Minplayers    string `xml:"minplayers"`
	Maxplayers    string `xml:"maxplayers"`
	Playingtime   string `xml:"playingtime"`
	Minplaytime   string `xml:"minplaytime"`
	Maxplaytime   string `xml:"maxplaytime"`
	Age           string `xml:"age"`
	Name          Names  `xml:"name"`
	Description   struct {
		Text string   `xml:",chardata"`
		Br   []string `xml:"br"`
	} `xml:"description"`
	Thumbnail               string `xml:"thumbnail"`
	Image                   string `xml:"image"`
	Boardgameaccessory      []Link `xml:"boardgameaccessory"`
	Boardgamepublisher      []Link `xml:"boardgamepublisher"`
	Boardgamepodcastepisode []Link `xml:"boardgamepodcastepisode"`
	Boardgamehonor          []Link `xml:"boardgamehonor"`
	Videogamebg             []Link `xml:"videogamebg"`
	Boardgamedesigner       []Link `xml:"boardgamedesigner"`
	Boardgameversion        []Link `xml:"boardgameversion"`
	Boardgamefamily         []Link `xml:"boardgamefamily"`
	Boardgameartist         []Link `xml:"boardgameartist"`
	Boardgamecategory       []Link `xml:"boardgamecategory"`
	Commerceweblink         []Link `xml:"commerceweblink"`
	Boardgamemechanic       []Link `xml:"boardgamemechanic"`
	Boardgamesubdomain      []Link `xml:"boardgamesubdomain"`
	Boardgameimplementation struct {
		Text     string `xml:",chardata"`
		Objectid string `xml:"objectid,attr"`
		Inbound  string `xml:"inbound,attr"`
	} `xml:"boardgameimplementation"`
	Poll       Polls      `xml:"poll"`
	Statistics Statistics `xml:"statistics"`
}

type Statistics struct {
	XMLName xml.Name `xml:"statistics"`
	Text    string   `xml:",chardata"`
	Page    string   `xml:"page,attr"`
	Ratings struct {
		Text         string `xml:",chardata"`
		Usersrated   string `xml:"usersrated"`
		Average      string `xml:"average"`
		Bayesaverage string `xml:"bayesaverage"`
		Ranks        struct {
			Text string `xml:",chardata"`
			Rank []struct {
				Text         string `xml:",chardata"`
				Type         string `xml:"type,attr"`
				ID           string `xml:"id,attr"`
				Name         string `xml:"name,attr"`
				Friendlyname string `xml:"friendlyname,attr"`
				Value        string `xml:"value,attr"`
				Bayesaverage string `xml:"bayesaverage,attr"`
			} `xml:"rank"`
		} `xml:"ranks"`
		Stddev        string `xml:"stddev"`
		Median        string `xml:"median"`
		Owned         string `xml:"owned"`
		Trading       string `xml:"trading"`
		Wanting       string `xml:"wanting"`
		Wishing       string `xml:"wishing"`
		Numcomments   string `xml:"numcomments"`
		Numweights    string `xml:"numweights"`
		Averageweight string `xml:"averageweight"`
	} `xml:"ratings"`
}

type Link struct {
	Text     string `xml:",chardata"`
	Objectid string `xml:"objectid,attr"`
}

const (
	EndpointGet = "boardgame/"
)

func (c *Client) Get(ctx context.Context, ids ...string) (*Boardgames, error) {
	u, err := url.Parse(c.XMLAddress + EndpointGet)
	if err != nil {
		return nil, err
	}
	u.Path += strings.Join(ids, ",")

	query := url.Values{}
	query.Set("stats", "1")

	u.RawQuery = query.Encode()

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

	decoder := xml.NewDecoder(resp.Body)
	out := &Boardgames{}
	err = decoder.Decode(out)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode, %w", err)
	}

	return out, nil
}

type Names []Name
type Name struct {
	Text      string `xml:",chardata"`
	Sortindex string `xml:"sortindex,attr"`
	Primary   string `xml:"primary,attr"`
}

func (ns Names) Principal() *Name {
	for _, n := range ns {
		if n.Primary == "true" {
			return &n
		}
	}
	return nil
}

type Polls []Poll

type Poll struct {
	Text       string `xml:",chardata"`
	Name       string `xml:"name,attr"`
	Title      string `xml:"title,attr"`
	Totalvotes string `xml:"totalvotes,attr"`
	Results    []struct {
		Text       string       `xml:",chardata"`
		Numplayers string       `xml:"numplayers,attr"`
		Result     []PollResult `xml:"result"`
	} `xml:"results"`
}
type PollResult struct {
	Text     string `xml:",chardata"`
	Value    string `xml:"value,attr"`
	Numvotes string `xml:"numvotes,attr"`
	Level    string `xml:"level,attr"`
}

func (pr PollResult) NumvotesInt() int {
	count, err := strconv.Atoi(pr.Numvotes)
	if err != nil {
		logrus.Warnf("Failed to parse Numvotes %s", err.Error())
		return 0
	}
	return count
}

func (p Polls) ByName(name string) Poll {
	for i := range p {
		if p[i].Name == name {
			return p[i]
		}
	}
	logrus.Warnf("Couldn't find Pool by name %s", name)
	return Poll{}
}

func (p Poll) SingleResult() PollResult {

	if len(p.Results) == 0 {
		return PollResult{}
	}
	result := PollResult{
		Numvotes: "0",
	}
	for _, r := range p.Results[0].Result {
		if r.NumvotesInt() > result.NumvotesInt() {
			result = r
		}
	}
	return result

}
