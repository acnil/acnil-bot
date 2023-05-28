package acnil

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

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

	btnAdmin = mainMenu.Text("👮 Administrador")

	adminMenu          = &tele.ReplyMarkup{ResizeKeyboard: true}
	btnForgotten       = adminMenu.Text("Juegos olvidados?")
	btnCancelAdminMenu = adminMenu.Text("Atrás")

	renameMenu      = &tele.ReplyMarkup{ResizeKeyboard: true}
	btnCancelRename = renameMenu.Text("Cancelar")

	startMenu = &tele.ReplyMarkup{ResizeKeyboard: true}
	btnStart  = renameMenu.Text("Empezar!")
)

// mainMenuReplyMarkup Given a member, builds the main menu keyboard with appropriate buttons.
func mainMenuReplyMarkup(member Member) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{ResizeKeyboard: true}

	std := []tele.Row{
		markup.Row(btnMyGames),
		markup.Row(btnEnGamonal, btnEnCentro),
		markup.Row(btnRename),
	}
	if member.Permissions == PermissionAdmin {
		std = append(std, markup.Row(btnAdmin))
	}
	markup.Reply(std...)
	markup.ResizeKeyboard = true
	return markup
}

func adminMenuReplyMarkup(member Member) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{ResizeKeyboard: true}
	markup.Reply(
		markup.Row(btnForgotten),
		markup.Row(btnCancelAdminMenu),
	)
	markup.ResizeKeyboard = true
	return markup

}

func init() {
	renameMenu.Reply(
		renameMenu.Row(btnCancelRename),
	)
	renameMenu.RemoveKeyboard = true

	startMenu.Reply(
		startMenu.Row(btnStart),
	)
	startMenu.RemoveKeyboard = true
}

// MembersDatabase gives access the the current member using the application
type MembersDatabase interface {
	Get(ctx context.Context, telegramID int64) (*Member, error)
	List(ctx context.Context) ([]Member, error)
	Append(ctx context.Context, member Member) error
	Update(ctx context.Context, member Member) error
}

// GameDatabase gives access to all the games registered in the excel
type GameDatabase interface {
	Find(ctx context.Context, name string) ([]Game, error)
	List(ctx context.Context) ([]Game, error)
	Get(ctx context.Context, id string, name string) (*Game, error)
	Update(ctx context.Context, game ...Game) error
}

// Sender sends something to telegram bot
type Sender interface {
	Send(to tele.Recipient, what interface{}, opts ...interface{}) (*tele.Message, error)
}

// ROAudit gives read only access to the audit database
type ROAudit interface {
	Find(ctx context.Context, query Query) ([]AuditEntry, error)
}

type Handler struct {
	MembersDB MembersDatabase
	GameDB    GameDatabase
	Audit     ROAudit
	Bot       Sender
}

func (h *Handler) Register(b *tele.Bot) {

	b.Handle("/start", h.Start)
	b.Handle(&btnStart, h.Start)

	b.Handle(tele.OnText, h.OnText)
	b.Handle("\ftake", h.OnTake)
	b.Handle("\ftake-all", h.OnTakeAll)
	b.Handle("\freturn", h.OnReturn)
	b.Handle("\freturn-all", h.OnReturnAll)
	b.Handle("\fmore", h.OnMore)
	b.Handle("\fauthorise", h.OnAuthorise)
	b.Handle("\fhistory", h.OnHistory)
	b.Handle("\fextendLease", h.OnExtendLease)
	b.Handle(&btnMyGames, h.MyGames)
	b.Handle(&btnEnGamonal, h.IsAuthorized(h.InGamonal))
	b.Handle(&btnEnCentro, h.IsAuthorized(h.InCentro))
	b.Handle(&btnRename, h.Rename)
	b.Handle(&btnCancelRename, h.CancelRename)

	b.Handle(&btnAdmin, h.OnAdmin)
	b.Handle(&btnForgotten, h.OnForgotten)

	h.Bot = b
}

// IsAuthorized find if the current user is registered in the application and the access has been accepted by an admin
// If a new user is detected, it will also emit an event for the admins and register it in the database.
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
			sender, _ := json.Marshal(c.Sender())
			log.WithField(ilog.FieldName, newMember.Nickname).WithField("sender", string(sender)).Info("Registering new user")
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

Si no lo recibes en 24h, avisa a @metalblueberry. 
`))
		}
		return next(c, *m)
	}
}

// Calls next if the user is admin, otherwise fallback to text handler
func (h Handler) IsAdmin(next func(tele.Context, Member) error) func(tele.Context, Member) error {
	return func(c tele.Context, m Member) error {
		if m.Permissions == PermissionAdmin {
			return next(c, m)
		}
		return h.onText(c, m)
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
	_, err = h.Bot.Send(newMember, "Ya tienes acceso! di /start o pulsa este botón para recibir el mensaje de bienvenida", startMenu)
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

Si algo va mal, habla con @MetalBlueberry`, member.Nickname), mainMenuReplyMarkup(member))
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

	lines := strings.Split(c.Text(), "\n")

	list := []Game{}
	for _, line := range lines {
		search, err := h.textSearchGame(log, c, member, line)
		if err != nil {
			log.WithError(err).Error("Failed to search games")
			return c.Send("No he podido buscar el juego, inténtalo otra vez")
		}
		list = append(list, search...)
	}

	if len(lines) == 1 {
		switch {
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
intenta darme el nombre concreto del juego que buscas. 
También puedes seleccionar un juego por su ID, solo dime el número de la lista.

Esto es todo lo que he encontrado`, mainMenu)
			for _, block := range SendList(list) {
				err := c.Send(block, mainMenuReplyMarkup(member))
				if err != nil {
					log.Error(err)
				}
			}
		}
	} else {
		selector := &tele.ReplyMarkup{}
		rows := []tele.Row{}
		rows = append(rows, selector.Row(
			selector.Data("Devolver todos", "return-all"),
		))
		rows = append(rows, selector.Row(
			selector.Data("Tomar prestados todos", "take-all"),
		))

		selector.Inline(rows...)

		for _, block := range SendList(list) {
			err := c.Send(block, selector)
			if err != nil {
				log.Error(err)
			}
		}
	}

	return nil
}

var mayBeAnID = regexp.MustCompile(`\d+\w*`)

func (h *Handler) textSearchGame(log *logrus.Entry, c tele.Context, member Member, text string) ([]Game, error) {
	list, err := h.GameDB.Find(context.TODO(), text)
	if err != nil {
		return nil, fmt.Errorf("Failed to find game by text, %w", err)
	}
	if len(list) > 1 {
		c.Send(fmt.Sprintf("He encontrado %d juegos con el nombre %s", len(list), text), mainMenuReplyMarkup(member))
		return list, nil
	}
	if len(list) == 1 {
		return list, nil
	}

	if mayBeAnID.MatchString(text) {
		id := strings.TrimLeft(text, "0")
		getResult, err := h.GameDB.Get(context.TODO(), id, "")
		if err != nil {
			if mmErr, ok := err.(MultipleMatchesError); ok {
				c.Send(fmt.Sprintf("Parece que hay varios juegos con el mismo ID %d", id), mainMenuReplyMarkup(member))
				return mmErr.Matches, nil
			}
			return nil, fmt.Errorf("failed to connect to GameDB, %w", err)
		}
		if getResult == nil {
			c.Send(fmt.Sprintf("No he encontrado ningún juego con el ID %d", id), mainMenuReplyMarkup(member))
			return []Game{}, nil
		}
		return []Game{*getResult}, nil
	}

	c.Send(fmt.Sprintf("No he podido encontrar ningún juego con el nombre %s", text), mainMenuReplyMarkup(member))

	return list, nil

}

func (h *Handler) OnTakeAll(c tele.Context) error {
	return h.IsAuthorized(h.onTakeAll)(c)
}

var gameLineMatch = regexp.MustCompile(`[🔴🟢]\s0*(.*?):\s(.*?)(\s\((.*)\))?$`)

func (h *Handler) onTakeAll(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Take All"), c.Sender())
	defer c.Respond()

	allGames, err := h.GameDB.List(context.Background())
	if err != nil {
		log.WithError(err).Error("Failed to get game from DB")
		c.Send("No he podido buscar el juego en la base de datos, inténtalo otra vez")
		return nil
	}

	games := Games{}
	hasBeenModified := false

	lines := strings.Split(c.Message().Text, "\n")
	for _, line := range lines {
		fragments := gameLineMatch.FindStringSubmatch(line)
		if len(fragments) != 5 {
			log.Errorf("Invalid number of matches, %s, [%s]", line, strings.Join(fragments, "|"))
			return c.Edit("Datos inválidos, vuelve a realizar la búsqueda")
		}
		id := fragments[1]
		name := fragments[2]
		expectedHolder := fragments[4]

		log.
			WithField("Game", name).
			WithField("ID", id)

		g, err := Games(allGames).Get(id, name)
		if err != nil {
			log.Info("Multiple matches for the game")
			return c.Send(fmt.Sprintf("Hay multiples coincidencias para el juego %s, %s.\n%s\nNo puedo realizar la operación", id, name, err.(MultipleMatchesError).Matches))
		}
		if g == nil {
			log.Info("Game not found")
			return c.Send(fmt.Sprintf("No he encontrado el juego %s: \"%s\", ¿Se ha modificado el excel? vuelve a darme la lista", id, name))
		}

		if g.Holder != expectedHolder {
			log.Info("It has been modified!")
			hasBeenModified = true
		}

		games = append(games, *g)
	}

	if hasBeenModified {
		log.Info("Detected conflict on TakeAll")
		c.Send("Parece que los datos han cambiado, revisa la información y vuelve a intentarlo")
		return h.bulk(c.Edit, games)
	}

	log.Info("Taking all games")
	for i := range games {
		log.
			WithField("Game", games[i].Name).
			WithField("ID", games[i].ID).
			Info("Taking game")
		games[i].Take(member.Nickname)
	}

	if err := h.GameDB.Update(context.Background(), games...); err != nil {
		log.WithError(err).Error("Failed to update gameDB")
		c.Send("No he podido actualizar la base de datos, vuelve a intentarlo")
	}

	return h.bulk(c.Edit, games)

}

func (h *Handler) OnTake(c tele.Context) error {
	return h.IsAuthorized(h.onTake)(c)
}

func (h *Handler) onTake(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Take"), c.Sender())

	g := NewGameFromData(c.Data())
	log = log.
		WithField("Game", g.Name).
		WithField("ID", g.ID)

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

	if !g.IsAvailable() {
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

	g.Take(member.Nickname)

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

func (h *Handler) OnReturnAll(c tele.Context) error {
	return h.IsAuthorized(h.onReturnAll)(c)
}

func (h *Handler) onReturnAll(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Return All"), c.Sender())
	defer c.Respond()

	allGames, err := h.GameDB.List(context.Background())
	if err != nil {
		log.WithError(err).Error("Failed to get game from DB")
		c.Send("No he podido buscar el juego en la base de datos, inténtalo otra vez")
		return nil
	}
	games := Games{}
	hasBeenModified := false

	lines := strings.Split(c.Message().Text, "\n")
	for _, line := range lines {
		fragments := gameLineMatch.FindStringSubmatch(line)
		if len(fragments) != 5 {
			log.Errorf("Invalid number of matches, %s, [%s]", line, strings.Join(fragments, "|"))
			return c.Edit("Datos inválidos, vuelve a realizar la búsqueda")
		}
		id := fragments[1]
		name := fragments[2]
		expectedHolder := fragments[4]

		log.
			WithField("Game", name).
			WithField("ID", id)

		g, err := Games(allGames).Get(id, name)
		if err != nil {
			log.Info("Multiple matches for the game")
			return c.Send(fmt.Sprintf("Hay multiples coincidencias para el juego %s, %s.\n%s\nNo puedo realizar la operación", id, name, err.(MultipleMatchesError).Matches))
		}
		if g == nil {
			log.Info("Game not found")
			return c.Send(fmt.Sprintf("No he encontrado el juego %s: \"%s\", ¿Se ha modificado el excel? vuelve a darme la lista", id, name))
		}

		if g.Holder != expectedHolder {
			log.Info("It has been modified!")
			hasBeenModified = true
		}

		games = append(games, *g)
	}

	if hasBeenModified {
		log.Info("Detected conflict on ReturnAll")
		c.Send("Parece que los datos han cambiado, revisa la información y vuelve a intentarlo")
		return h.bulk(c.Edit, games)
	}

	log.Info("Returning all games")
	for i := range games {
		log.
			WithField("Game", games[i].Name).
			WithField("ID", games[i].ID).
			Info("Return game")
		games[i].Return()
	}

	if err := h.GameDB.Update(context.Background(), games...); err != nil {
		log.WithError(err).Error("Failed to update gameDB")
		c.Send("No he podido actualizar la base de datos, vuelve a intentarlo")
	}

	return h.bulk(c.Edit, games)
}

func (h *Handler) OnReturn(c tele.Context) error {
	return h.IsAuthorized(h.onReturn)(c)
}

func (h *Handler) onReturn(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Return"), c.Sender())

	g := NewGameFromData(c.Data())
	log = log.
		WithField("Game", g.Name).
		WithField("ID", g.ID)

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

	if !g.IsHeldBy(member) && member.Permissions != PermissionAdmin {
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

	g.Return()

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

func (h *Handler) OnHistory(c tele.Context) error {
	return h.IsAuthorized(h.onHistory)(c)
}

type PrettyAuditEntry struct {
	AuditEntry
	TimeFormat string
}

func (p PrettyAuditEntry) String() string {
	if p.AuditEntry.Holder == "" {
		return fmt.Sprintf("%s: %s", p.Timestamp.Format(p.TimeFormat), "devuelto")
	}
	return fmt.Sprintf("%s: %s", p.Timestamp.Format(p.TimeFormat), p.Holder)
}

func (h *Handler) onHistory(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "History"), c.Sender())

	defer c.Respond()

	g := NewGameFromData(c.Data())
	log = log.WithField("Game", g.Name)

	entries, err := h.Audit.Find(context.TODO(), Query{
		Game:  &g,
		Limit: 100,
	})
	if err != nil {
		log.WithError(err).Error("Error finding audit history for game")
		return c.Send("Wops! No he podido encontrar el historial.... Díselo a @MetalBlueberry para que lo arregle")
	}

	prettyEntries := make([]PrettyAuditEntry, 0, len(entries))
	previous := AuditEntry{}
	for _, e := range entries {
		if e.Holder == previous.Holder {
			continue
		}
		previous = e
		prettyEntries = append(prettyEntries, PrettyAuditEntry{
			AuditEntry: e,
			TimeFormat: "2006-01-02",
		})
	}

	if len(prettyEntries) == 0 {
		return c.Send("Parece que nadie ha usado este juego nunca.\nPuedes ser el primero!")
	}

	for _, block := range SendList(prettyEntries) {
		err := c.Send(block, mainMenuReplyMarkup(member))
		if err != nil {
			log.Error(err)
		}
	}
	return nil
}

func (h *Handler) OnExtendLease(c tele.Context) error {
	return h.IsAuthorized(h.onExtendLease)(c)
}

func (h *Handler) onExtendLease(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "OnExtendLease"), c.Sender())

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

	if member.Permissions != PermissionAdmin && !g.IsHeldBy(member) {
		err := c.Edit("Parece que alguien ha modificado los datos. te envío los últimos actualizados")
		if err != nil {
			log.Print(err)
		}
		err = c.Send(g.Card(), g.Buttons(member))
		if err != nil {
			log.Print(err)
		}
		log.Info("Conflict on ExtendLease")
		return c.Respond()
	}

	if g.TakeDate.IsZero() {
		c.Respond()
		return c.Send("Necesito la fecha de prestamos para poder añadir mas dias")
	}

	g.SetLeaseTimeDays(g.LeaseDays() + 21)
	if g.IsLeaseExpired() {
		log.Errorf("Lease is still expired!!")
	}

	err = h.GameDB.Update(context.TODO(), g)
	if err != nil {
		c.Edit(err.Error())
		log.Error("Failed to update game database")
		return c.Respond()
	}

	err = c.Edit(g.Card(), g.Buttons(member))
	if err != nil {
		log.Errorf("Failed to edit card, %s", err)
	}
	log.Info("Game lease time extended")
	return c.Respond()
}

func (h *Handler) MyGames(c tele.Context) error {
	return h.IsAuthorized(h.myGames)(c)
}

func (h *Handler) myGames(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "MyGames"), c.Sender())

	if member.Nickname == "" {
		c.Send("Wops! Parece que no tienen ningún nombre")
		return h.rename(c, member)
	}

	gameList, err := h.GameDB.List(context.TODO())
	if err != nil {
		return c.Send(err.Error())
	}

	myGames := []Game{}
	for _, game := range gameList {
		if game.IsHeldBy(member) {
			myGames = append(myGames, game)
		}
	}

	if len(myGames) == 0 {
		log.Info("Not games found for the user")
		return c.Send("No tienes ningún juego a tu nombre")
	}

	log.Info("Sending list to user")
	return h.bulk(c.Send, myGames)
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
		err := c.Send(block, mainMenuReplyMarkup(member))
		if err != nil {
			log.Error(err)
		}
	}

	return nil
}

// Bulk sends a message with bulk operations for the list of games
func (h *Handler) bulk(action func(what interface{}, opts ...interface{}) error, games []Game) error {
	selector := &tele.ReplyMarkup{}
	rows := []tele.Row{}

	if Games(games).CanReturn() {
		rows = append(rows, selector.Row(
			selector.Data("Devolver todos", "return-all"),
		))
	}
	if Games(games).CanTake() {
		rows = append(rows, selector.Row(
			selector.Data("Tomar prestados todos", "take-all"),
		))
	}

	selector.Inline(rows...)

	for _, block := range SendList(games) {
		err := action(block, selector)
		if err != nil {
			return err
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

	return c.Send("Okey, Te llamas "+member.Nickname, mainMenuReplyMarkup(member))
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

	if newName == member.Nickname {
		member.State = ""
		return c.Send("Okey, te dejo el mismo nombre", mainMenuReplyMarkup(member))
	}

	member.State = ""
	member.Nickname = newName
	c.Send("Listo! ahora te llamas "+member.Nickname, mainMenuReplyMarkup(member))
	return nil
}

func (h *Handler) OnAdmin(c tele.Context) error {
	return h.IsAuthorized(h.IsAdmin(h.onAdmin))(c)
}

func (h *Handler) onAdmin(c tele.Context, member Member) error {
	return c.Send("This is the future admin section", adminMenuReplyMarkup(member))
}

func (h *Handler) OnForgotten(c tele.Context) error {
	return h.IsAuthorized(h.IsAdmin(h.onForgotten))(c)
}

func (h *Handler) onForgotten(c tele.Context, member Member) error {

	games, err := h.GameDB.List(context.Background())
	if err != nil {
		c.Send("Wops! Algo ha ido mal!")
		return c.Send(err.Error())
	}

	forgottenGames := []Game{}
	for _, g := range games {
		if !g.IsAvailable() && g.IsLeaseExpired() {
			forgottenGames = append(forgottenGames, g)
		}
	}

	sort.Slice(forgottenGames, func(i, j int) bool { return forgottenGames[i].LeaseDays() < forgottenGames[j].LeaseDays() })

	for _, g := range forgottenGames {
		c.Send(g.Card(), g.Buttons(member))
	}

	return nil
}
