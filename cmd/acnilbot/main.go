package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	tele "gopkg.in/telebot.v3"
)

const (
	TstSheetID = "1WFJL0tNJ21XtHfy6ksJvRe5rSUeTG44BmQbeEWYo4Y0"
)

func main() {

	credentialsFile := GetEnv("CREDENTIALS_FILE", "credentials.json")
	sheetID := os.Getenv("SHEET_ID")
	if sheetID == "" {
		log.Fatal("SHEET_ID must be defined")
	}

	botToken := os.Getenv("TOKEN")
	if botToken == "" {
		log.Fatal("TOKEN must be defined")
	}

	pref := tele.Settings{
		Token:  botToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	srv, err := acnil.CreateClientFromCredentals(context.TODO(), credentialsFile)
	if err != nil {
		panic(err)
	}
	db := acnil.GameDatabase{
		SRV:       srv,
		ReadRange: "A:L",
		Sheet:     "Juegos de mesa",
		SheetID:   sheetID,
	}

	b.Handle("/start", func(c tele.Context) error {
		log.Println(c.Text())
		return c.Send(`Bienvenido al bot de Acnil,
Por ahora puedo ayudarte a tomar prestados y devolver los juegos de forma mas sencilla. Simplemente mándame el nombre del juego y yo lo buscaré por ti.

También puede buscar por parte del nombre. por ejemplo, Intenta decir "Exploding"


Recuerda que estoy en pruebas y no utilizo datos reales, puedes ver el excel en el siguiente link.
https://docs.google.com/spreadsheets/d/1WFJL0tNJ21XtHfy6ksJvRe5rSUeTG44BmQbeEWYo4Y0/edit#gid=0

Recuerda que el bot utiliza tu nombre de usuario de telegram para identificarte, cunando reserves un juego lo utilizará como tu nombre y solo te dejará devolver juegos que tengas tu.

Si algo va mal, habla con @MetalBlueberry`)
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		if c.Message().FromGroup() {
			log.Print(c.Chat())
			log.Println("skip group")
			return nil
		}

		log.Println(c.Text())

		if c.Sender().Username == "" {
			return c.Send("El bot necesita que tengas un nombre de usuario definido. Puedes elegir uno desde la configuración de tu perfil de Telegram")
		}

		_, err := strconv.Atoi(c.Text())
		if err == nil {
			getResult, err := db.Get(context.TODO(), c.Text(), "")
			if err != nil {
				c.Send(err.Error())
				return c.Respond()
			}
			if getResult == nil {
				c.Send("No he podido encontrar un juego con ese ID")
				return c.Respond()
			}

			err = c.Send(getResult.Card(), getResult.Buttons(c))
			if err != nil {
				log.Println(err)
			}
			return c.Respond()
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

		getResult, err := db.Get(context.TODO(), g.ID, g.Name)
		if err != nil {
			c.Edit(err.Error())
			return c.Respond()
		}
		if getResult == nil {
			c.Edit("No he podido encontrar el juego. Intenta volver a buscarlo, tal vez se ha modificado el excel")
			return c.Respond()
		}

		g = *getResult

		if g.Holder != "" {
			err := c.Edit("Parece que alguien ha modificado los datos, te envío los últimos actualizados")
			if err != nil {
				log.Print(err)
			}
			err = c.Send(g.Card(), g.Buttons(c))
			if err != nil {
				log.Print(err)
			}
			return c.Respond()
		}
		g.Holder = c.Sender().Username

		err = db.Update(context.TODO(), g)
		if err != nil {
			c.Edit(err.Error())
			return c.Respond()
		}

		c.Edit(g.Card(), g.Buttons(c))
		return c.Respond()
	})
	b.Handle("\freturn", func(c tele.Context) error {
		log.Println("return")
		g := acnil.NewGameFromData(c.Data())

		getResult, err := db.Get(context.TODO(), g.ID, g.Name)
		if err != nil {
			c.Edit(err.Error())
			return c.Respond()
		}
		if getResult == nil {
			c.Edit("No he podido encontrar el juego. Intenta volver a buscarlo, tal vez se ha modificado el excel")
			return c.Respond()
		}

		g = *getResult

		if g.Holder != c.Sender().Username {
			err := c.Edit("Parece que alguien ha modificado los datos, te envío los últimos actualizados")
			if err != nil {
				log.Print(err)
			}
			err = c.Send(g.Card(), g.Buttons(c))
			if err != nil {
				log.Print(err)
			}
			return c.Respond()
		}

		g.Holder = ""

		err = db.Update(context.TODO(), g)
		if err != nil {
			c.Edit(err.Error())
			return c.Respond()
		}

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

func GetEnv(key string, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}
