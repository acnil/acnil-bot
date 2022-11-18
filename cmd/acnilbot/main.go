package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/metalblueberry/acnil-bot/pkg/acnil"
	"github.com/metalblueberry/acnil-bot/pkg/ilog"
	"github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

var (
	mainMenu = &tele.ReplyMarkup{ResizeKeyboard: true}
	// Reply buttons.
	btnMyGames   = mainMenu.Text("🎲 Mis Juegos")
	btnEnGamonal = mainMenu.Text("Lista de Gamonal")
	btnEnCentro  = mainMenu.Text("Lista del Centro")
)

func main() {

	credentialsFile := GetEnv("CREDENTIALS_FILE", "credentials.json")
	sheetID := os.Getenv("SHEET_ID")
	if sheetID == "" {
		logrus.Fatal("SHEET_ID must be defined")
	}

	botToken := os.Getenv("TOKEN")
	if botToken == "" {
		logrus.Fatal("TOKEN must be defined")
	}

	pref := tele.Settings{
		Token:  botToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		logrus.Fatal(err)
		return
	}

	srv, err := acnil.CreateClientFromCredentals(context.TODO(), credentialsFile)
	if err != nil {
		panic(err)
	}
	handler := &Handler{
		MembersDB: acnil.NewMembersDatabase(srv, sheetID),
		GameDB: &acnil.GameDatabase{
			SRV:       srv,
			ReadRange: "A:L",
			Sheet:     "Juegos de mesa",
			SheetID:   sheetID,
		},
	}

	b.Handle("/start", handler.Start)
	b.Handle(tele.OnText, handler.OnText)
	b.Handle("\ftake", handler.OnTake)
	b.Handle("\freturn", handler.OnReturn)
	b.Handle("\fmore", handler.OnMore)
	b.Handle(&btnMyGames, handler.MyGames)
	b.Handle(&btnEnGamonal, handler.EnGamonal)
	b.Handle(&btnEnCentro, handler.EnCentro)

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

// func IsAuthorized(bot *tele.Bot, groupID int64) func(user *tele.User) error {
// 	return func(user *tele.User) error {
// 		chat, err := bot.ChatByID(groupID)
// 		if err != nil {
// 			return fmt.Errorf("Failed to get group by id, %w", err)
// 		}
// 		chatMember, err := bot.ChatMemberOf(chat, user)
// 		if err != nil {
// 			log.Info()
// 			return fmt.Errorf("Failed to get chat member of, %w", err)
// 		}
// 		switch chatMember.Role {
// 		case tele.Administrator, tele.Creator, tele.Member:
// 			return nil
// 		}
// 		return fmt.Errorf("No tienes permiso en este grupo")
// 	}
// }

type Handler struct {
	MembersDB *acnil.MembersDatabase
	GameDB    *acnil.GameDatabase
}

func (h *Handler) IsAuthorized(log *logrus.Entry, c tele.Context) (*acnil.Member, error) {
	m, err := h.MembersDB.Get(context.Background(), c.Sender().ID)
	if err != nil {
		log.WithError(err).Error("Cannot check membersDB")
		return nil, c.Send(fmt.Sprintf("Algo ha ido mal..., %s", err.Error()))
	}
	if m == nil {
		newMember := acnil.NewMemberFromTelegram(c.Sender())
		log.WithField(ilog.FieldName, newMember.Nickname).Info("Registering new user")
		m = &newMember
		h.MembersDB.Append(context.Background(), newMember)
	}

	if !strings.EqualFold(m.Permissions, "si") {
		log.
			WithField(ilog.FieldName, m.Nickname).
			Info("Permission denied")
		return nil, c.Send(fmt.Sprintf("Hola, Antes de nada, has de ir al documento de inventario.\nEn la pestaña de miembros habrá aparecido tu nombre al final.\nTienes que cambiar tus permisos para poder empezar a usar este bot\n\nCuando tengas permiso, vuelve a enviar /start para recibir instrucciones"))
	}
	return m, nil
}

func (h *Handler) Start(c tele.Context) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Start"), c.Sender())
	log.Info(c.Text())

	member, err := h.IsAuthorized(log, c)
	if err != nil {
		return err
	}
	if member == nil {
		return nil
	}

	mainMenu.Reply(
		mainMenu.Row(btnMyGames),
		mainMenu.Row(btnEnGamonal, btnEnCentro),
	)

	return c.Send(`Bienvenido al bot de Acnil,
Por ahora puedo ayudarte a tomar prestados y devolver los juegos de forma mas sencilla. Simplemente mándame el nombre del juego y yo lo buscaré por ti.

También puedo buscar por parte del nombre. por ejemplo, Intenta decir "Exploding"

Por último, si me mandas el ID de un juego, también puedo encontrarlo.

Si algo va mal, habla con @MetalBlueberry`, mainMenu)

}

func (h *Handler) OnText(c tele.Context) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Text"), c.Sender())

	if c.Message().FromGroup() {
		log.WithField("Chat", c.Chat().FirstName).Debug("skip group")
		return nil
	}

	member, err := h.IsAuthorized(log, c)
	if err != nil {
		log.WithError(err).Error("Unable to authorize")
		return err
	}
	if member == nil {
		return nil
	}

	log = log.WithField("Text", c.Text())

	_, err = strconv.Atoi(c.Text())
	if err == nil {
		getResult, err := h.GameDB.Get(context.TODO(), c.Text(), "")
		if err != nil {
			log.WithError(err).Error("Failed to connect to GameDB")
			return c.Send(err.Error())
		}
		if getResult == nil {
			log.Info("Unable to find game by ID")
			return c.Send("No he podido encontrar un juego con ese ID")
		}
		log.WithField("Game", getResult.Name).Info("Found Game by ID")
		return c.Send(getResult.Card(), getResult.Buttons(member))
	}

	list, err := h.GameDB.Find(context.TODO(), c.Text())
	if err != nil {
		panic(err)
	}

	switch {
	case len(list) == 0:
		log.Info("Unable to find game")
		return c.Send("No he podido encontrar ningún juego con ese nombre")
	case len(list) <= 3:
		for _, g := range list {
			log.WithField("Game", g.Name).Info("Found Game")
			err := c.Send(g.Card(), g.Buttons(member))
			if err != nil {
				log.Error(err)
			}
		}
	default:
		log.WithField("count", len(list)).Info("Found multiple games")
		c.Send("He encontrado varios juegos, intenta darme mas detalles del juego que buscas. Esta es una lista de todo lo que he encontrado")
		for _, block := range SendList(list) {
			err := c.Send(block)
			if err != nil {
				log.Error(err)
			}
		}
	}

	return nil
}

func (h *Handler) OnTake(c tele.Context) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Take"), c.Sender())

	member, err := h.IsAuthorized(log, c)
	if err != nil {
		return err
	}
	if member == nil {
		return nil
	}

	g := acnil.NewGameFromData(c.Data())
	log = log.WithField("Game", g.Name)

	getResult, err := h.GameDB.Get(context.TODO(), g.ID, g.Name)
	if err != nil {
		log.WithError(err).Error("Unable to get game from DB")
		c.Edit(err.Error())
		return c.Respond()
	}
	if getResult == nil {
		log.Warn("Unable to find game")
		c.Edit("No he podido encontrar el juego. Intenta volver a buscarlo, tal vez se ha modificado el excel")
		return c.Respond()
	}

	g = *getResult

	if g.Holder != "" {
		err := c.Edit("Parece que alguien ha modificado los datos, te envío los últimos actualizados")
		if err != nil {
			log.Error(err)
		}
		err = c.Send(g.Card(), g.Buttons(member))
		if err != nil {
			log.Error(err)
		}
		log.Info("Conflict on Take")
		return c.Respond()
	}
	g.Holder = member.Nickname

	err = h.GameDB.Update(context.TODO(), g)
	if err != nil {
		c.Edit(err.Error())
		log.Error("Failed to update game database")
		return c.Respond()
	}

	c.Edit(g.Card(), g.Buttons(member))
	log.Info("Game taken")
	return c.Respond()
}

func (h *Handler) OnReturn(c tele.Context) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Return"), c.Sender())

	member, err := h.IsAuthorized(log, c)
	if err != nil {
		return err
	}
	if member == nil {
		return nil
	}

	g := acnil.NewGameFromData(c.Data())
	log = log.WithField("Game", g.Name)

	getResult, err := h.GameDB.Get(context.TODO(), g.ID, g.Name)
	if err != nil {
		log.WithError(err).Error("Unable to get from GameDB")
		c.Edit(err.Error())
		return c.Respond()
	}
	if getResult == nil {
		log.Warn("Unable to find game")
		c.Edit("No he podido encontrar el juego. Intenta volver a buscarlo, tal vez se ha modificado el excel")
		return c.Respond()
	}

	g = *getResult

	if g.Holder != member.Nickname {
		err := c.Send("Parece que alguien ha modificado los datos, desde la última vez. te envío los últimos actualizados")
		if err != nil {
			log.Print(err)
		}
		err = c.Edit(g.Card(), g.Buttons(member))
		if err != nil {
			log.Print(err)
		}
		log.Info("Conflict on Return")
		return c.Respond()
	}

	g.Holder = ""

	err = h.GameDB.Update(context.TODO(), g)
	if err != nil {
		c.Edit(err.Error())
		log.Error("Failed to update game database")
		return c.Respond()
	}

	c.Edit(g.Card(), g.Buttons(member))
	log.Info("Game returned")
	return c.Respond()
}

func (h *Handler) OnMore(c tele.Context) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "More"), c.Sender())

	member, err := h.IsAuthorized(log, c)
	if err != nil {
		return err
	}
	if member == nil {
		return nil
	}

	g := acnil.NewGameFromData(c.Data())
	log = log.WithField("Game", g.Name)

	getResult, err := h.GameDB.Get(context.Background(), g.ID, g.Name)
	if err != nil {
		c.Edit(err.Error())
		log.WithError(err).Error("Unable to get from GameDB")
		return c.Respond()
	}
	g = *getResult

	if getResult == nil {
		c.Edit("No he podido encontrar el juego. Intenta volver a buscarlo, tal vez se ha modificado el excel")
		log.Warn("Unable to find game")
		return c.Respond()
	}

	err = c.Edit(g.MoreCard(), g.Buttons(member))
	if err != nil {
		log.WithError(err).Error("Failed to send message")
	}
	log.Info("Details requested")
	return c.Respond()
}

func (h *Handler) MyGames(c tele.Context) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "MyGames"), c.Sender())

	member, err := h.IsAuthorized(log, c)
	if err != nil {
		return err
	}
	if member == nil {
		return nil
	}

	gameList, err := h.GameDB.List(context.TODO())
	if err != nil {
		return c.Send(err.Error())
	}

	myGames := []acnil.Game{}
	for _, game := range gameList {
		if game.Holder == member.Nickname {
			myGames = append(myGames, game)
		}
	}

	if len(myGames) == 0 {
		return c.Send("No tienes ningun juego a tu nombre")
	}

	for _, g := range myGames {
		log.WithField("Game", g.Name).Info("Found Game owned by user")
		err := c.Send(g.Card(), g.Buttons(member))
		if err != nil {
			log.Error(err)
		}
	}

	return nil
}

func (h *Handler) EnCentro(c tele.Context) error {
	return h.inLocation(c, "Centro")
}

func (h *Handler) EnGamonal(c tele.Context) error {
	return h.inLocation(c, "Gamonal")
}

func (h *Handler) inLocation(c tele.Context, location string) error {
	log := ilog.WithTelegramUser(logrus.
		WithField(ilog.FieldHandler, "inLocation").
		WithField(ilog.FieldLocation, location),
		c.Sender())

	member, err := h.IsAuthorized(log, c)
	if err != nil {
		return err
	}
	if member == nil {
		return nil
	}

	gameList, err := h.GameDB.List(context.TODO())
	if err != nil {
		return c.Send(err.Error())
	}

	inLocation := []acnil.Game{}
	for _, game := range gameList {
		if strings.EqualFold(game.Location, location) {
			inLocation = append(inLocation, game)
		}
	}

	if len(inLocation) == 0 {
		return c.Send("No se han encontrado juegos")
	}

	for _, block := range SendList(inLocation) {
		err := c.Send(block)
		if err != nil {
			log.Error(err)
		}
	}

	return nil
}
