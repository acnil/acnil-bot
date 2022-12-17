package acnil

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	btnRename    = mainMenu.Text("🧍 Cambiar Nombre")

	renameMenu      = &tele.ReplyMarkup{ResizeKeyboard: true}
	btnCancelRename = renameMenu.Text("Cancelar")
)

func addMainMenuReplyMarkup(rm *tele.ReplyMarkup) {
	rm.Reply(
		rm.Row(btnMyGames),
		rm.Row(btnEnGamonal, btnEnCentro),
		rm.Row(btnRename),
	)
	rm.ResizeKeyboard = true
	rm.Inline()
}

func init() {
	addMainMenuReplyMarkup(mainMenu)
	renameMenu.Reply(
		renameMenu.Row(btnCancelRename),
	)
	renameMenu.RemoveKeyboard = true
}

type MembersDatabase interface {
	Get(ctx context.Context, telegramID int64) (*Member, error)
	List(ctx context.Context) ([]Member, error)
	Append(ctx context.Context, member Member) error
	Update(ctx context.Context, member Member) error
}

type GameDatabase interface {
	Find(ctx context.Context, name string) ([]Game, error)
	List(ctx context.Context) ([]Game, error)
	Get(ctx context.Context, id string, name string) (*Game, error)
	Update(ctx context.Context, game Game) error
}

type Sender interface {
	Send(to tele.Recipient, what interface{}, opts ...interface{}) (*tele.Message, error)
}

type Handler struct {
	MembersDB MembersDatabase
	GameDB    GameDatabase
	Bot       Sender
}

func (h *Handler) Register(b *tele.Bot) {

	b.Handle("/start", h.Start)
	b.Handle(tele.OnText, h.OnText)
	b.Handle("\ftake", h.OnTake)
	b.Handle("\freturn", h.OnReturn)
	b.Handle("\fmore", h.OnMore)
	b.Handle("\fauthorise", h.OnAuthorise)
	b.Handle(&btnMyGames, h.IsAuthorized(h.MyGames))
	b.Handle(&btnEnGamonal, h.IsAuthorized(h.InGamonal))
	b.Handle(&btnEnCentro, h.IsAuthorized(h.InCentro))
	b.Handle(&btnRename, h.Rename)
	b.Handle(&btnCancelRename, h.CancelRename)
	h.Bot = b
}

func (h *Handler) IsAuthorized(next func(tele.Context, Member) error) func(tele.Context) error {
	return func(c tele.Context) error {
		log := ilog.WithTelegramUser(
			logrus.WithField(ilog.FieldHandler, "Authorization"),
			c.Sender(),
		)
		m, err := h.MembersDB.Get(context.Background(), c.Sender().ID)
		if err != nil {
			log.WithError(err).Error("Cannot check membersDB")
			return c.Send(fmt.Sprintf("Algo ha ido mal..., %s", err.Error()))
		}
		if m == nil {
			newMember := NewMemberFromTelegram(c.Sender())
			log.WithField(ilog.FieldName, newMember.Nickname).Info("Registering new user")
			m = &newMember
			h.MembersDB.Append(context.Background(), newMember)
			h.notifyAdminsOfNewLogin(log, newMember)
		}

		if !m.Permissions.IsAuthorised() {
			log.
				WithField(ilog.FieldName, m.Nickname).
				Info("Permission denied")
			return c.Send(fmt.Sprintf(`Hola,
He notificado a un administrador de que necesitas acceso. Te avisaré cuando lo tengas.

También puedes hacerlo tu mismo.
Has de ir al documento de inventario.  En la pestaña de miembros habrá aparecido tu nombre al final. Tienes que cambiar tus permisos para poder empezar a usar este bot.

Cuando tengas permiso a PermissionYes, vuelve a enviar /start para recibir instrucciones`))
		}
		return next(c, *m)
	}
}

func (h *Handler) notifyAdminsOfNewLogin(log *logrus.Entry, newMember Member) error {
	members, err := h.MembersDB.List(context.Background())
	if err != nil {
		return err
	}

	selector := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	rows = append(rows, selector.Row(
		selector.Data("Dar acceso", "authorise", newMember.TelegramID),
	))
	selector.Inline(rows...)

	for _, m := range members {
		if m.Permissions == PermissionAdmin {
			log.WithField("Admin", m.Nickname).Info("Notifying admin")
			_, err = h.Bot.Send(&m, fmt.Sprintf("Nuevo usuario %s registrado", newMember.Nickname), selector)
			if err != nil {
				log.WithError(err).Error("Failed to notify admin")
			}
		}
	}
	return nil
}

func (h *Handler) OnAuthorise(c tele.Context) error {
	return h.IsAuthorized(h.onAuthorise)(c)
}

func (h *Handler) onAuthorise(c tele.Context, _ Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Authorise"), c.Sender())

	newMemberID, err := strconv.Atoi(c.Data())
	if err != nil {
		c.Edit("No he podido leer el ID de usuario, " + err.Error())
		return nil
	}
	newMember, err := h.MembersDB.Get(context.Background(), int64(newMemberID))
	if err != nil {
		c.Send("Inténtalo de nuevo, " + err.Error())
		return err
	}

	if newMember == nil {
		c.Send("Parece que el nuevo usuario no está en el excel")
		return nil
	}

	newMember.Permissions = PermissionYes
	err = h.MembersDB.Update(context.Background(), *newMember)
	if err != nil {
		c.Send("Parece que algo ha ido mal, " + err.Error())
		return nil
	}
	_, err = h.Bot.Send(newMember, "Ya tienes acceso! di /start para recibir el mensaje de bienvenida")
	if err != nil {
		log.Errorf("Error sending message to new member, %s", err)
	}

	return c.Edit(fmt.Sprintf("Se ha dado acceso al usuario %s de forma correcta", newMember.Nickname))
}

func (h *Handler) Start(c tele.Context) error {
	return h.IsAuthorized(h.start)(c)
}

func (h *Handler) start(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Start"), c.Sender())
	log.Info(c.Text())

	return c.Send(fmt.Sprintf(`Bienvenido al bot de Acnil,
Tu nombre es %s y se ha generado en base a tu nombre de Telegram. Es el nombre que aparecerá en el excel cuando reserves un juego. Si quieres cambiarlo, utiliza el teclado a continuación.

Por ahora puedo ayudarte a tomar prestados y devolver los juegos de forma mas sencilla. Simplemente mándame el nombre del juego y yo lo buscaré por ti.

También puedo buscar por parte del nombre. por ejemplo, Intenta decir "Exploding"

Por último, si me mandas el ID de un juego, también puedo encontrarlo.

Si algo va mal, habla con @MetalBlueberry`, member.Nickname), mainMenu)
}

func (h *Handler) skipGroup(next func(c tele.Context) error) func(c tele.Context) error {
	return func(c tele.Context) error {
		if c.Message().FromGroup() {
			log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "GroupSkip"), c.Sender())
			log.WithField("Chat", c.Chat().FirstName).Debug("skip group")
			return nil
		}
		return next(c)
	}
}

func (h *Handler) OnText(c tele.Context) error {
	return h.IsAuthorized(h.onText)(c)
}

func (h *Handler) onText(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Text"), c.Sender())

	log = log.WithField(ilog.FieldText, c.Text())

	switch member.State {
	case "rename":
		return h.onRename(c, member)
	}

	id, err := strconv.Atoi(c.Text())
	if err == nil {
		getResult, err := h.GameDB.Get(context.TODO(), strconv.Itoa(id), "")
		if err != nil {
			log.WithError(err).Error("Failed to connect to GameDB")
			return c.Send(err.Error(), mainMenu)
		}
		if getResult == nil {
			log.Info("Unable to find game by ID")
			return c.Send("No he podido encontrar un juego con ese ID", mainMenu)
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
		return c.Send("No he podido encontrar ningún juego con ese nombre", mainMenu)
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
		c.Send(`He encontrado varios juegos, 
intenta darme mas detalles del juego que buscas. 
También puedes seleccionar un juego por su ID, solo dime el número de la lista.

Esto es todo lo que he encontrado`, mainMenu)
		for _, block := range SendList(list) {
			err := c.Send(block, mainMenu)
			if err != nil {
				log.Error(err)
			}
		}
	}

	return nil
}

func (h *Handler) OnTake(c tele.Context) error {
	return h.IsAuthorized(h.onTake)(c)
}

func (h *Handler) onTake(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Take"), c.Sender())

	g := NewGameFromData(c.Data())
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
	g.TakeDate = time.Now()

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
	return h.IsAuthorized(h.onReturn)(c)
}

func (h *Handler) onReturn(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Return"), c.Sender())

	g := NewGameFromData(c.Data())
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
		err := c.Edit("Parece que alguien ha modificado los datos. te envío los últimos actualizados")
		if err != nil {
			log.Print(err)
		}
		err = c.Send(g.Card(), g.Buttons(member))
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
	return h.IsAuthorized(h.onMore)(c)
}

func (h *Handler) onMore(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "More"), c.Sender())

	g := NewGameFromData(c.Data())
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

func (h *Handler) MyGames(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "MyGames"), c.Sender())

	gameList, err := h.GameDB.List(context.TODO())
	if err != nil {
		return c.Send(err.Error())
	}

	myGames := []Game{}
	for _, game := range gameList {
		if game.Holder == member.Nickname {
			myGames = append(myGames, game)
		}
	}

	if len(myGames) == 0 {
		return c.Send("No tienes ningún juego a tu nombre")
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

func (h *Handler) InCentro(c tele.Context, member Member) error {
	return h.inLocation(c, member, "Centro")
}

func (h *Handler) InGamonal(c tele.Context, member Member) error {
	return h.inLocation(c, member, "Gamonal")
}

func (h *Handler) inLocation(c tele.Context, member Member, location string) error {
	log := ilog.WithTelegramUser(logrus.
		WithField(ilog.FieldHandler, "inLocation").
		WithField(ilog.FieldLocation, location),
		c.Sender())

	gameList, err := h.GameDB.List(context.TODO())
	if err != nil {
		return c.Send(err.Error())
	}

	inLocation := []Game{}
	for _, game := range gameList {
		if strings.EqualFold(game.Location, location) {
			inLocation = append(inLocation, game)
		}
	}

	if len(inLocation) == 0 {
		return c.Send("No se han encontrado juegos")
	}

	for _, block := range SendList(inLocation) {
		err := c.Send(block, mainMenu)
		if err != nil {
			log.Error(err)
		}
	}

	return nil
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

func (h *Handler) Rename(c tele.Context) error {
	return h.IsAuthorized(h.rename)(c)
}

func (h *Handler) rename(c tele.Context, member Member) error {
	member.State = "rename"

	err := h.MembersDB.Update(context.Background(), member)
	if err != nil {
		return err
	}
	c.Send("Dime el nombre que quieres tener, Tu nombre actual es "+member.Nickname, renameMenu)
	return nil
}

func (h *Handler) CancelRename(c tele.Context) error {
	return h.IsAuthorized(h.cancelRename)(c)
}

func (h *Handler) cancelRename(c tele.Context, member Member) error {
	member.State = ""
	err := h.MembersDB.Update(context.Background(), member)
	if err != nil {
		return err
	}

	return c.Send("Okey, Te llamas "+member.Nickname, mainMenu)
}

func (h *Handler) onRename(c tele.Context, member Member) error {
	defer func() {
		err := h.MembersDB.Update(context.Background(), member)
		if err != nil {
			c.Send(err.Error())
		}
	}()

	newName := strings.TrimSpace(c.Text())
	if len(newName) > 25 {
		return c.Send("No puedes usar un nombre tan largo...", renameMenu)
	}

	members, err := h.MembersDB.List(context.Background())
	if err != nil {
		c.Send(err.Error())
		return err
	}

	if newName == member.Nickname {
		member.State = ""
		return c.Send("Okey, te dejo el mismo nombre", mainMenu)
	}

	for _, other := range members {
		if other.Nickname == newName {
			return c.Send("Wops! Parece que ya hay otra persona usando este nombre.", renameMenu)
		}
	}

	member.State = ""
	member.Nickname = newName
	c.Send("Listo! ahora te llamas "+member.Nickname, mainMenu)
	return nil
}
