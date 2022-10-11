package main

import (
	"context"
	"log"
	"os"

	"github.com/metalblueberry/acnil-bot/pkg/bgg"
)

func main() {

	c := bgg.NewClient()
	resp, err := c.Search(context.Background(), os.Args[1])
	if err != nil {
		panic(err)
	}

	for _, i := range resp.Items {
		log.Println(i.Href)
	}
}
