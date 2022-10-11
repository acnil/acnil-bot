package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	tele "gopkg.in/telebot.v3"
)

const (
	SheetID = "***REMOVED***"
)

func main() {
	pref := tele.Settings{
		Token:  os.Getenv("TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	srv, err := acnil.CreateClientFromCredentals(context.TODO(), "credentials.json")
	if err != nil {
		panic(err)
	}
	db := acnil.GameDatabase{
		SRV:       srv,
		ReadRange: "A:J",
		Sheet:     "Juegos de mesa",
		SheetID:   SheetID,
	}

	var (
	// Universal markup builders.

	// Inline buttons.
	//
	// Pressing it will cause the client to
	// send the bot a callback.
	//
	// Make sure Unique stays unique as per button kind
	// since it's required for callback routing to work.
	//
	)

	b.Handle("/start", func(c tele.Context) error {
		log.Println(c.Text())
		return c.Send(`Bienvenido al bot de Acnil,
Por ahora puedo ayudarte a tomar prestados y devolver los juegos de forma mas sencilla. Simplemente mándame el nombre del juego y yo lo buscaré por ti.

También puede buscar por parte del nombre. por ejemplo, Intenta decir "Exploding"


Recuerda que estoy en pruebas y no utilizo datos reales, puedes ver el excel en el siguiente link.
https://docs.google.com/spreadsheets/d/***REMOVED***/edit#gid=0

Recuerda que el bot utiliza tu nombre de usuario de telegram para identificarte, cunando reserves un juego lo utilizará como tu nombre y solo te dejará devolver juegos que tengas tu.

Si algo va mal, habla con @MetalBlueberry`)
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		log.Println(c.Text())

		if c.Sender().Username == "" {
			return c.Send("El bot necesita que tengas un nombre de usuario definido. Puedes elegir uno desde la configuración de tu perfil de Telegram")
		}

		list, err := db.Find(context.TODO(), c.Text())
		if err != nil {
			panic(err)
		}

		switch {
		case len(list) == 0:
			return c.Send("No he podido encontrar ningún juego con ese nombre")
		case len(list) <= 3:
			for _, g := range list {
				err := c.Send(g.Card(), g.Buttons(c))
				if err != nil {
					log.Print(err)
				}
			}
		default:
			c.Send("He encontrado varios juegos, intenta darme mas detalles del juego que buscas. Esta es una lista de todo lo que he encontrado")
			for _, block := range SendList(list) {
				err := c.Send(block)
				if err != nil {
					log.Print(err)
				}
			}
		}

		return nil
	})

	b.Handle("\ftake", func(c tele.Context) error {
		log.Println("take")
		g := acnil.NewGameFromData(c.Data())
		g.Holder = c.Sender().Username

		err := db.Update(context.TODO(), g)
		if err != nil {
			c.Edit(err.Error())
			return c.Respond()
		}

		list, err := db.Get(context.TODO(), g.Name)
		if err != nil {
			c.Edit(err.Error())
			return c.Respond()
		}
		if len(list) == 0 {
			c.Edit("No he podido encontrar el juego, inténtalo de nuevo")
			return c.Respond()
		}

		if len(list) != 1 {
			c.Edit("Wops! Parece que hay mas de un juego con este nombre, modifica el excel manualmente para asegurar que no hay nombres identicos.")
			return c.Respond()
		}
		g = list[0]

		c.Edit(g.Card(), g.Buttons(c))
		return c.Respond()
	})
	b.Handle("\freturn", func(c tele.Context) error {
		log.Println("return")
		g := acnil.NewGameFromData(c.Data())
		g.Holder = ""

		err := db.Update(context.TODO(), g)
		if err != nil {
			c.Edit(err.Error())
			return c.Respond()
		}

		list, err := db.Get(context.TODO(), g.Name)
		if err != nil {
			c.Edit(err.Error())
			return c.Respond()
		}
		if len(list) == 0 {
			c.Edit("No he podido encontrar el juego, inténtalo de nuevo")
			return c.Respond()
		}

		if len(list) != 1 {
			c.Edit("Wops! Parece que hay mas de un juego con este nombre, modifica el excel manualmente para asegurar que no hay nombres identicos.")
			return c.Respond()
		}
		g = list[0]

		c.Edit(g.Card(), g.Buttons(c))
		return c.Respond()
	})

	b.Start()
}

func SendList[T fmt.Stringer](items []T) []string {
	msgCharacters := 0
	msgFragments := make([]string, 0)
	msgs := make([]string, 0)

	for _, item := range items {
		line := item.String()

		msgCharacters = msgCharacters + len(line)

		if msgCharacters >= 3900 { // Max Telegram Message Length
			msgCharacters = len(line)
			msgs = append(msgs, strings.Join(msgFragments, "\n"))
			msgFragments = msgFragments[0:0]
		}
		msgFragments = append(msgFragments, line)
	}
	if len(msgFragments) > 0 {
		msgs = append(msgs, strings.Join(msgFragments, "\n"))
	}
	return msgs
}
