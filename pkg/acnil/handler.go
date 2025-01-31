package acnil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	httplambda "github.com/acnil/acnil-bot/pkg/httpLambda"
	"github.com/acnil/acnil-bot/pkg/ilog"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	tele "gopkg.in/telebot.v3"
)

var (
	mainMenu = &tele.ReplyMarkup{ResizeKeyboard: true}
	// Reply buttons.
	btnMyGames          = mainMenu.Text("🎲 Mis Juegos")
	btnEnGamonal        = mainMenu.Text("Lista de Gamonal")
	btnEnCentro         = mainMenu.Text("Lista del Centro")
	btnRename           = mainMenu.Text("🧍 Cambiar Nombre")
	btnJuegatron        = mainMenu.Text("Juegatron!")
	btnExitJuegatron    = mainMenu.Text("Salir de Juegatron")
	btnListJuegatron    = mainMenu.Text("Lista de Juegatron")
	cancelJuegatronMenu = &tele.ReplyMarkup{ResizeKeyboard: true}
	btnCancelJuegatron  = cancelJuegatronMenu.Text("Cancelar préstamo")

	btnAdmin = mainMenu.Text("👮 Administrador")

	adminMenu           = &tele.ReplyMarkup{ResizeKeyboard: true}
	btnForgotten        = adminMenu.Text("Juegos olvidados?")
	btnNotInAnyPlace    = adminMenu.Text("Juegos en ningún sitio")
	btnGamesTakenByUser = adminMenu.Text("Juegos cogidos por usuario")
	btnCancelAdminMenu  = adminMenu.Text("Atrás")

	cancelMenu = &tele.ReplyMarkup{ResizeKeyboard: true}
	btnCancel  = cancelMenu.Text("Cancel")

	startMenu = &tele.ReplyMarkup{ResizeKeyboard: true}
	btnStart  = startMenu.Text("Empezar!")
)

// mainMenuReplyMarkup Given a member, builds the main menu keyboard with appropriate buttons.
func mainMenuReplyMarkup(member Member) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{ResizeKeyboard: true}

	std := []tele.Row{
		markup.Row(btnMyGames),
		markup.Row(btnEnGamonal, btnEnCentro),
		markup.Row(btnRename),
		// markup.Row(btnJuegatron),
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
		markup.Row(btnNotInAnyPlace),
		markup.Row(btnGamesTakenByUser),
		markup.Row(btnCancelAdminMenu),
	)
	markup.ResizeKeyboard = true
	return markup

}

func juegatronReplyMarkup() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{ResizeKeyboard: true}
	markup.Reply(
		markup.Row(btnExitJuegatron),
		markup.Row(btnListJuegatron),
	)
	markup.ResizeKeyboard = true
	return markup

}

func init() {
	cancelMenu.Reply(
		cancelMenu.Row(btnCancel),
	)
	cancelMenu.RemoveKeyboard = true

	startMenu.Reply(
		startMenu.Row(btnStart),
	)
	startMenu.RemoveKeyboard = true

	cancelJuegatronMenu.Reply(
		cancelJuegatronMenu.Row(btnCancelJuegatron),
	)
	cancelJuegatronMenu.RemoveKeyboard = true
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

	JuegatronGameDB ROGameDatabase
	JuegatronAudit  *JuegatronAudit

	Bot Sender
}

func AttatchLambdaContext(next tele.HandlerFunc) tele.HandlerFunc {
	return func(ctx tele.Context) error {
		update := ctx.Update()
		ctx.Set(strconv.Itoa(update.ID), httplambda.GetContext(update.ID))
		return next(ctx)
	}
}

func (h *Handler) Register(handlerGroup *tele.Group) {
	handlerGroup.Use(AttatchLambdaContext)
	handlerGroup.Use(OnlyPrivateChatMiddleware)

	handlerGroup.Handle("/start", h.Start)
	handlerGroup.Handle(&btnStart, h.Start)

	handlerGroup.Handle(tele.OnText, h.OnText)
	handlerGroup.Handle("\ftake", h.OnTake)
	handlerGroup.Handle("\ftake-all", h.OnTakeAll)
	handlerGroup.Handle("\freturn", h.OnReturn)
	handlerGroup.Handle("\freturn-all", h.OnReturnAll)
	handlerGroup.Handle("\fmore", h.OnMore)
	handlerGroup.Handle("\fauthorise", h.OnAuthorise)
	handlerGroup.Handle("\fhistory", h.OnHistory)
	handlerGroup.Handle("\fextendLease", h.OnExtendLease)
	handlerGroup.Handle("\fgame-page-1", h.OnGamePage(1))
	handlerGroup.Handle("\fgame-page-2", h.OnGamePage(2))
	handlerGroup.Handle("\fswitch-location", h.OnSwitchLocation)
	handlerGroup.Handle("\fupdate-comment", h.OnUpdateCommentButton)
	handlerGroup.Handle(&btnMyGames, h.MyGames)
	handlerGroup.Handle(&btnEnGamonal, h.IsAuthorized(h.InGamonal))
	handlerGroup.Handle(&btnEnCentro, h.IsAuthorized(h.InCentro))
	handlerGroup.Handle(&btnRename, h.Rename)
	handlerGroup.Handle(&btnJuegatron, h.OnJuegatron)
	handlerGroup.Handle(&btnExitJuegatron, h.OnExitJuegatron)
	handlerGroup.Handle(&btnListJuegatron, h.OnListJuegatron)
	handlerGroup.Handle("\fjuegatron-return", h.OnJuegatronReturn)
	handlerGroup.Handle("\fundo-juegatron-return", h.OnUndoJuegatronReturn)
	handlerGroup.Handle("\fjuegatron-take", h.OnJuegatronTake)
	handlerGroup.Handle(&btnCancelJuegatron, h.OnCancelJuegatron)

	handlerGroup.Handle(&btnCancel, h.Cancel)

	handlerGroup.Handle(&btnAdmin, h.OnAdmin)
	handlerGroup.Handle(&btnCancelAdminMenu, h.OnCancelAdminMenu)
	handlerGroup.Handle(&btnForgotten, h.OnForgotten)
	handlerGroup.Handle(&btnNotInAnyPlace, h.OnNotInAnyPlace)
	handlerGroup.Handle(&btnGamesTakenByUser, h.OnGamesTakenByUser)
}

func OnlyPrivateChatMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(ctx tele.Context) error {
		if ctx.Chat().Type == tele.ChatPrivate {
			next(ctx)
		}
		return nil
	}

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
			err := h.MembersDB.Append(context.Background(), newMember)
			if err != nil {
				logrus.
					WithError(err).
					Error("Failed to append member to database")
			}
			err = h.notifyAdminsOfNewLogin(log, newMember)
			if err != nil {
				logrus.
					WithError(err).
					Error("Failed to notify admins of new login")
			}
		}

		// Migration code, this makes sure all users have the telegram name and the telegramUsername set
		if m.TelegramName == "" {
			m.TelegramName = fmt.Sprintf("%s %s", c.Sender().FirstName, c.Sender().LastName)
			m.TelegramUsername = c.Sender().Username
			err := h.MembersDB.Update(context.Background(), *m)
			if err != nil {
				logrus.
					WithError(err).
					Error("Failed to update member in database")
			}
		}

		if !m.Permissions.IsAuthorised() {
			log.
				WithField(ilog.FieldName, m.Nickname).
				Info("Permission denied")
			return c.Send(fmt.Sprintf(`Hola,
He notificado a un administrador de que necesitas acceso. Te avisaré cuando lo tengas.

Si no lo recibes en 24h, avisa a @MetalBlueberry. 
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
	switch {
	case member.State.Is(StateActionJuegatron):
		return h.onJuegatronText(c, member)
	case member.State.Is(StateActionJuegatronWaitingForName):
		return h.onJuegatronTakeWaitForName(c, member)
	case isACommandForSure.Match([]byte(c.Text())):
		return h.onSearchByText(c, member)
	case member.State.Is(StateActionRename):
		return h.onRename(c, member)
	case member.State.Is(StateActionUpdateComment):
		return h.onUpdateComment(c, member)
	case member.State.Is(StateGetGamesTakenByUser):
		return h.onGetGamesTakenByUser(c, member)
	default:
		return h.onSearchByText(c, member)
	}
}

func GetContext(c tele.Context) (_ context.Context, cancelFunc func()) {
	ctx, ok := c.Get(strconv.Itoa(c.Update().ID)).(context.Context)
	if !ok {
		logrus.Info("using background")
		ctx = context.Background()
	}

	logrus.Info("using default deadline")
	return context.WithDeadline(ctx, time.Now().Add(5*time.Second))
}

func (h *Handler) onSearchByText(c tele.Context, member Member) error {
	ctx, cancel := GetContext(c)
	defer cancel()

	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Text"), c.Sender())

	log = log.WithField(ilog.FieldText, c.Text())

	gameList, err := h.GameDB.List(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to cache game data")
		return c.Send("Wops! Algo ha ido mal, vuelve a intentarlo en unos momentos.\n" + err.Error())
	}

	lines := strings.Split(c.Text(), "\n")

	list := Games{}
	for _, line := range lines {
		search, err := h.textSearchGame(log, c, gameList, line, mainMenuReplyMarkup(member))
		if err != nil {
			log.WithError(err).Error("Failed to search games")
			return c.Send("No he podido buscar el juego, inténtalo otra vez")
		}
		list = append(list, search...)
		if ctx.Err() != nil {
			return c.Send(fmt.Sprintf("Wops! Parece que no me ha dado tiempo... Envíame algo mas simple.\n%s", ctx.Err()))
		}
	}

	if len(lines) == 1 {
		switch {
		case len(list) <= 3:
			for _, g := range list {
				log.
					WithField("Game", g.Name).
					Info("Found Game")
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
				if ctx.Err() != nil {
					return c.Send(fmt.Sprintf("Wops! Parece que no me ha dado tiempo... Envíame algo mas simple.\n%s", ctx.Err()))
				}
			}
		}
	} else {
		duplicate, list := list.FindDuplicates()
		if len(duplicate) > 0 {
			c.Send("Parece que los siguientes elementos están duplicados en la lista que me has pasado")
			for _, block := range SendList(duplicate) {
				err := c.Send(block)
				if err != nil {
					log.Error(err)
				}
				if ctx.Err() != nil {
					return c.Send(fmt.Sprintf("Wops! Parece que no me ha dado tiempo... Envíame algo mas simple.\n%s", ctx.Err()))
				}
			}
		}

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
			if ctx.Err() != nil {
				return c.Send(fmt.Sprintf("Wops! Parece que no me ha dado tiempo... Envíame algo mas simple.\n%s", ctx.Err()))
			}
		}
	}

	return nil
}

var mayBeAnID = regexp.MustCompile(`^[/]?0*(\d+\w*)$`)
var isAnIDForSure = regexp.MustCompile(`^[/]?(\d+)$`)
var isACommandForSure = regexp.MustCompile(`^/.*$`)

func (h *Handler) textSearchGame(log *logrus.Entry, c tele.Context, gameList Games, text string, markup *tele.ReplyMarkup) ([]Game, error) {
	if gameLineMatch.MatchString(text) {
		game, err := NewGameFromLine(text)
		if err != nil {
			log.WithError(err).Error("Failed to convert line to a game")
			c.Send("Esto no debería ocurrir, avisa a victor")
			return []Game{}, nil
		}

		g, err := gameList.Get(game.ID, game.Name)
		if err != nil {
			if mmErr, ok := err.(MultipleMatchesError); ok {
				c.Send(fmt.Sprintf("Parece que hay varios juegos con el mismo ID %s Nombre %s", game.ID, game.Name))
				return mmErr.Matches, nil
			}
			return []Game{}, nil
		}
		if g == nil {
			c.Send(fmt.Sprintf("No he encontrado ningún juego con ID %s, nombre %s", game.ID, game.Name), markup)
			return []Game{}, nil
		}
		return []Game{*g}, nil
	}

	if !isAnIDForSure.MatchString(text) {
		list := gameList.Find(text)
		if len(list) > 1 {
			c.Send(fmt.Sprintf("He encontrado %d juegos con el nombre \"%s\"", len(list), text), markup)
			return list, nil
		}
		if len(list) == 1 {
			return list, nil
		}
	}

	if mayBeAnID.MatchString(text) {
		id := mayBeAnID.FindStringSubmatch(text)[1]
		getResult, err := gameList.Get(id, "")
		if err != nil {
			if mmErr, ok := err.(MultipleMatchesError); ok {
				c.Send(fmt.Sprintf("Parece que hay varios juegos con el mismo ID %s", id), markup)
				return mmErr.Matches, nil
			}
			return nil, fmt.Errorf("failed to connect to GameDB, %w", err)
		}
		if getResult == nil {
			c.Send(fmt.Sprintf("No he encontrado ningún juego con el ID %s", id), markup)
			return []Game{}, nil
		}
		return []Game{*getResult}, nil
	}

	c.Send(fmt.Sprintf("No he podido encontrar ningún juego con el nombre %s", text), markup)
	return []Game{}, nil

}

func (h *Handler) OnTakeAll(c tele.Context) error {
	return h.IsAuthorized(h.onTakeAll)(c)
}

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
		lineGame, err := NewGameFromLine(line)
		if err != nil {
			log.WithError(err).Errorf("Invalid line")
			return c.Edit("Datos inválidos, vuelve a realizar la búsqueda")
		}

		log.
			WithField("Game", lineGame.Name).
			WithField("ID", lineGame.ID)

		g, err := Games(allGames).Get(lineGame.ID, lineGame.Name)
		if err != nil {
			log.Info("Multiple matches for the game")
			return c.Send(fmt.Sprintf("Hay multiples coincidencias para el juego %s, %s.\n%s\nNo puedo realizar la operación", lineGame.ID, lineGame.Name, err.(MultipleMatchesError).Matches))
		}
		if g == nil {
			log.Info("Game not found")
			return c.Send(fmt.Sprintf("No he encontrado el juego %s: \"%s\", ¿Se ha modificado el excel? vuelve a darme la lista", lineGame.ID, lineGame.Name))
		}

		if g.Holder != lineGame.Holder {
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

	g, err := NewGameFromCard(c.Message().Text)
	if err != nil {
		c.Edit("Wops! Algo ha ido mal....\nInténtalo de nuevo")
		return fmt.Errorf("failed to load data form card, %w", err)
	}

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
		lineGame, err := NewGameFromLine(line)
		if err != nil {
			log.WithError(err).Errorf("Invalid line")
			return c.Edit("Datos inválidos, vuelve a realizar la búsqueda")
		}

		log.
			WithField("Game", lineGame.Name).
			WithField("ID", lineGame.ID)

		g, err := Games(allGames).Get(lineGame.ID, lineGame.Name)
		if err != nil {
			log.Info("Multiple matches for the game")
			return c.Send(fmt.Sprintf("Hay multiples coincidencias para el juego %s, %s.\n%s\nNo puedo realizar la operación", lineGame.ID, lineGame.Name, err.(MultipleMatchesError).Matches))
		}
		if g == nil {
			log.Info("Game not found")
			return c.Send(fmt.Sprintf("No he encontrado el juego %s: \"%s\", ¿Se ha modificado el excel? vuelve a darme la lista", lineGame.ID, lineGame.Name))
		}

		if g.Holder != lineGame.Holder {
			log.
				WithField("CurrentHolder", g.Holder).
				WithField("ExpectedHolder", lineGame.Holder).
				Info("It has been modified!")
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

	g, err := NewGameFromCard(c.Message().Text)
	if err != nil {
		c.Edit("Wops! Algo ha ido mal....\nInténtalo de nuevo")
		return fmt.Errorf("failed to load data form card, %w", err)
	}
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

	if g.IsAvailable() {
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

	g, err := NewGameFromCard(c.Message().Text)
	if err != nil {
		c.Edit("Wops! Algo ha ido mal....\nInténtalo de nuevo")
		return fmt.Errorf("failed to load data form card, %w", err)
	}

	log = log.
		WithField("Game", g.Name).
		WithField("ID", g.ID)

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

type HolderChanged struct {
	AuditEntry
	TimeFormat string
}
type CommentChanged struct {
	AuditEntry
	TimeFormat string
}

type AuditTmplData struct {
	Game     Game
	Holders  []HolderChanged
	Comments []CommentChanged
}

var auditTmpl = template.Must(template.New("audit").Parse(`
Prestamos del juego:
/{{.Game.ID}}:{{.Game.Name}}

{{ range .Holders -}} 
{{ .Timestamp.Format .TimeFormat }}: {{if .Holder}}🔴 {{.Holder}}{{ else }}🟢 {{.Location }}{{end}}
{{ end }}
{{ if .Comments -}} 
Comentarios:
{{ range .Comments -}} 
{{ .Timestamp.Format .TimeFormat }}:
{{if .Comments}}{{ .Comments }}{{else}}Comentario eliminado{{end}}
{{ end }}
{{ end }}
`))

func (h *Handler) onHistory(c tele.Context, member Member) error {
	ctx, cancel := GetContext(c)
	defer cancel()

	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "History"), c.Sender())

	defer c.Respond()

	g, err := NewGameFromCard(c.Message().Text)
	if err != nil {
		c.Edit("Wops! Algo ha ido mal....\nInténtalo de nuevo")
		return fmt.Errorf("failed to load data form card, %w", err)
	}

	log = log.
		WithField("Game", g.Name).
		WithField("ID", g.ID)

	entries, err := h.Audit.Find(ctx, Query{
		Game:  &g,
		Limit: 100,
	})
	if err != nil {
		log.WithError(err).Error("Error finding audit history for game")
		return c.Send("Wops! No he podido encontrar el historial.... Díselo a @MetalBlueberry para que lo arregle")
	}

	log.WithField("auditLength", len(entries)).Info("Found audit events")

	holderChanged := make([]HolderChanged, 0, len(entries))
	previous := AuditEntry{}
	for _, e := range entries {
		if e.Holder == previous.Holder {
			continue
		}
		previous = e
		holderChanged = append(holderChanged, HolderChanged{
			AuditEntry: e,
			TimeFormat: "2006-01-02",
		})
		if ctx.Err() != nil {
			return c.Send("Wops! No me ha dado tiempo... avisa a @MetalBlueberry, es posible que el fichero sea demasiado grande")
		}
	}

	commentChanged := make([]CommentChanged, 0, len(entries))
	previous = AuditEntry{}
	for _, e := range entries {
		if e.Comments == previous.Comments {
			continue
		}
		previous = e
		commentChanged = append(commentChanged, CommentChanged{
			AuditEntry: e,
			TimeFormat: "2006-01-02",
		})
		if ctx.Err() != nil {
			return c.Send("Wops! No me ha dado tiempo... avisa a @MetalBlueberry, es posible que el fichero sea demasiado grande")
		}
	}

	if len(holderChanged) == 0 {
		return c.Send("Parece que nadie ha usado este juego nunca.\nPuedes ser el primero!")
	}

	buf := &bytes.Buffer{}
	auditTmpl.Execute(buf, AuditTmplData{
		Game:     g,
		Holders:  holderChanged,
		Comments: commentChanged,
	})

	return c.Send(buf.String())
}

func (h *Handler) OnExtendLease(c tele.Context) error {
	return h.IsAuthorized(h.onExtendLease)(c)
}

func (h *Handler) onExtendLease(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "OnExtendLease"), c.Sender())

	g, err := NewGameFromCard(c.Message().Text)
	if err != nil {
		c.Edit("Wops! Algo ha ido mal....\nInténtalo de nuevo")
		return fmt.Errorf("failed to load data form card, %w", err)
	}

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
	return h.inLocation(c, member, LocationCentro)
}

func (h *Handler) InGamonal(c tele.Context, member Member) error {
	return h.inLocation(c, member, LocationGamonal)
}

func (h *Handler) inLocation(c tele.Context, member Member, location Location) error {
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
		if game.IsInLocation(location) {
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
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "Rename"), c.Sender())
	member.State.SetRename()

	err := h.MembersDB.Update(context.Background(), member)
	if err != nil {
		log.WithError(err).Error("Failed to update memberDB")
		return err
	}
	c.Send("Dime el nombre que quieres tener, Tu nombre actual es "+member.Nickname, cancelMenu)
	log.Info("Rename action requested")
	return nil
}

func (h *Handler) Cancel(c tele.Context) error {
	return h.IsAuthorized(h.cancelRename)(c)
}

func (h *Handler) cancelRename(c tele.Context, member Member) error {
	member.State.Clear()
	err := h.MembersDB.Update(context.Background(), member)
	if err != nil {
		return err
	}

	return c.Send("Okey, Cancelado.", mainMenuReplyMarkup(member))
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
		return c.Send("No puedes usar un nombre tan largo...", cancelMenu)
	}

	if newName == member.Nickname {
		member.State.Clear()
		return c.Send("Okey, te dejo el mismo nombre", mainMenuReplyMarkup(member))
	}

	member.State.Clear()
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

func (h *Handler) OnCancelAdminMenu(c tele.Context) error {
	return h.IsAuthorized(h.IsAdmin(h.onCancelAdminMenu))(c)
}

func (h *Handler) onCancelAdminMenu(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "CancelAdminMenu"), c.Sender())
	member.State.Clear()
	h.MembersDB.Update(context.Background(), member)
	log.Info("Going back")
	return c.Send("Volviendo al menu principal", mainMenuReplyMarkup(member))
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

func (h *Handler) OnNotInAnyPlace(c tele.Context) error {
	return h.IsAuthorized(h.IsAdmin(h.onNotInAnyPlace))(c)
}

func (h *Handler) onNotInAnyPlace(c tele.Context, member Member) error {
	games, err := h.GameDB.List(context.Background())
	if err != nil {
		c.Send("Wops! Algo ha ido mal!")
		return c.Send(err.Error())
	}

	notInAnyPlace := []Game{}
	for _, g := range games {
		if !g.IsInLocation(LocationGamonal) && !g.IsInLocation(LocationCentro) {
			notInAnyPlace = append(notInAnyPlace, g)
		}
	}

	sort.Slice(notInAnyPlace, func(i, j int) bool { return notInAnyPlace[i].LeaseDays() < notInAnyPlace[j].LeaseDays() })

	for _, g := range notInAnyPlace {
		c.Send(g.Card(), g.Buttons(member))
	}

	if len(notInAnyPlace) == 0 {
		c.Send("Todo está bien")
	}

	return nil
}

func (h *Handler) OnGamePage(page int) func(c tele.Context) error {
	return func(c tele.Context) error {
		return h.IsAuthorized(h.onGamePage(page))(c)
	}
}

func (h *Handler) onGamePage(page int) func(c tele.Context, member Member) error {
	return func(c tele.Context, member Member) error {
		log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "GamePage"), c.Sender()).WithField(ilog.FieldPage, page)

		g, err := NewGameFromCard(c.Message().Text)
		if err != nil {
			c.Edit("Wops! Algo ha ido mal....\nInténtalo de nuevo")
			return fmt.Errorf("failed to load data form card, %w", err)
		}
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

		err = c.Edit(g.Card(), g.ButtonsForPage(member, page))
		if err != nil {
			log.Errorf("Failed to edit card, %s", err)
		}
		log.Info("Display page")
		return c.Respond()
	}
}

func (h *Handler) OnSwitchLocation(c tele.Context) error {
	return h.IsAuthorized(h.onSwitchLocation)(c)
}

func (h *Handler) onSwitchLocation(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "SwitchLocation"), c.Sender())

	g, err := NewGameFromCard(c.Message().Text)
	if err != nil {
		c.Edit("Wops! Algo ha ido mal....\nInténtalo de nuevo")
		return fmt.Errorf("failed to load data form card, %w", err)
	}

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

	switch {
	case g.IsInLocation(LocationCentro):
		g.Location = string(LocationGamonal)
	case g.IsInLocation(LocationGamonal):
		g.Location = string(LocationCentro)
	default:
		log.Warn("Failed to determine current location, moving to Gamonal")
		g.Location = string(LocationGamonal)
	}
	log = log.WithField(ilog.FieldLocation, g.Location)

	err = h.GameDB.Update(context.TODO(), g)
	if err != nil {
		c.Edit(err.Error())
		log.Error("Failed to update game database")
		return c.Respond()
	}

	err = c.Edit(g.Card(), g.ButtonsForPage(member, 2))
	if err != nil {
		log.WithError(err).Error("Failed to update card")
	}
	log.Info("Switched location")

	return c.Respond()
}

func (h *Handler) OnUpdateCommentButton(c tele.Context) error {
	return h.IsAuthorized(h.onUpdateCommentButton)(c)
}

func (h *Handler) onUpdateCommentButton(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "UpdateCommentButton"), c.Sender())

	g, err := NewGameFromCard(c.Message().Text)
	if err != nil {
		c.Edit("Wops! Algo ha ido mal....\nInténtalo de nuevo")
		return fmt.Errorf("failed to load data form card, %w", err)
	}

	log = log.WithField("Game", g.Name)

	c.Send(fmt.Sprintf("Dime que comentario quieres dejar para el juego %s: %s", g.ID, g.Name), cancelMenu)

	member.State.SetUpdateComment(g)

	err = h.MembersDB.Update(context.Background(), member)
	if err != nil {
		log.Error("Failed to updated memberDB")
		return err
	}

	return c.Respond()
}

func (h *Handler) OnCancelCommentButton(c tele.Context) error {
	return h.IsAuthorized(h.onCancelCommentButton)(c)
}

func (h *Handler) onCancelCommentButton(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "CancelUpdateCommentButton"), c.Sender())
	defer c.Respond()

	member.State.Clear()

	err := h.MembersDB.Update(context.Background(), member)
	if err != nil {
		log.Error("Failed to updated memberDB")
		return err
	}
	return c.Send("Ok, no dejo ningún comentario", mainMenuReplyMarkup(member))

}

func (h *Handler) onUpdateComment(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "UpdateComment"), c.Sender())

	g := NewGameFromLineData(member.State.Data)

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

	member.State.Clear()

	err = h.MembersDB.Update(context.Background(), member)
	if err != nil {
		log.Error("Failed to updated memberDB")
		return c.Respond()
	}

	g.Comments = c.Text()

	err = h.GameDB.Update(context.Background(), g)
	if err != nil {
		log.Error("Failed to update game DB")
	}

	c.Send(g.Card(), g.Buttons(member))

	return c.Respond()
}

func (h *Handler) OnGamesTakenByUser(c tele.Context) error {
	return h.IsAuthorized(h.IsAdmin(h.onGamesTakenByUser))(c)
}

func (h *Handler) onGamesTakenByUser(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "GamesTakenByUser"), c.Sender())
	games, err := h.GameDB.List(context.Background())
	if err != nil {
		c.Send("Wops! algo ha ido mal. Vuelve a intentarlo mas tarde")
		return fmt.Errorf("Failed to load game list, %w", err)
	}

	names := []string{}
	for _, g := range games {
		if !g.IsAvailable() {
			names = append(names, g.Holder)
		}
	}
	slices.Sort(names)
	uniqNames := slices.Compact(names)

	markup := &tele.ReplyMarkup{ResizeKeyboard: true}

	base := []tele.Row{
		markup.Row(btnCancelAdminMenu),
	}

	for _, name := range uniqNames {
		base = append(base, tele.Row{markup.Text(name)})
	}

	markup.ResizeKeyboard = true
	markup.Reply(base...)

	member.State.SetGetGamesTakenByUser()
	err = h.MembersDB.Update(context.Background(), member)
	if err != nil {
		c.Send("Wops! algo ha ido mal, inténtalo de nuevo")
		return fmt.Errorf("Failed to update membersDB, %w", err)
	}
	log.Info("Ready to return games taken by user")
	return c.Send("Dime la persona que quieres o utiliza el teclado", markup)

}

type GetGamesTakenByUserTemplateData struct {
	Entries []struct {
		Name      string
		Timestamp time.Time
		ID        string
	}
	TimeFormat string
}

func (h *Handler) onGetGamesTakenByUser(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "GetGamesTakenByUser"), c.Sender())

	entries, err := h.Audit.Find(context.Background(), Query{
		Member: &Member{
			Nickname: c.Text(),
		},
	})
	if err != nil {
		c.Send("Wops! Algo ha ido mal, vuelve a intentarlo mas tarde")
		return fmt.Errorf("Failed to get audit, %w", err)
	}
	if len(entries) == 0 {
		log.Info("No games found")
		return c.Send("No he encontrado juegos para este nombre. ¿Seguro que este miembro existe?")
	}

	data := GetGamesTakenByUserTemplateData{
		TimeFormat: "2006-01-02",
	}

	for _, entry := range entries {
		if len(data.Entries) == 0 {
			timestamp := entry.TakeDate
			if timestamp.IsZero() {
				timestamp = entry.Timestamp
			}
			data.Entries = append(data.Entries, struct {
				Name      string
				Timestamp time.Time
				ID        string
			}{
				Name:      entry.Name,
				Timestamp: timestamp,
				ID:        entry.ID,
			})
			continue
		}
		lastEntry := data.Entries[len(data.Entries)-1]
		if lastEntry.Timestamp != entry.TakeDate || lastEntry.Name != entry.Name {
			timestamp := entry.TakeDate
			if timestamp.IsZero() {
				timestamp = entry.Timestamp
			}
			data.Entries = append(data.Entries, struct {
				Name      string
				Timestamp time.Time
				ID        string
			}{
				Name:      entry.Name,
				Timestamp: timestamp,
				ID:        entry.ID,
			})
		}
	}

	tpl, _ := tmpl.Parse(`
{{- $g := . -}}
{{ range .Entries -}}
{{ .Timestamp.Format $g.TimeFormat }}: /{{.ID}} {{ .Name }}
{{ end }}`)

	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, data)
	if err != nil {
		return c.Send(err.Error())
	}

	log.WithField("member", c.Text()).Info("Served list for User")
	return c.Send(buf.String())

}

func (h *Handler) OnJuegatron(c tele.Context) error {
	return h.IsAuthorized(h.onJuegatron)(c)
}

func (h *Handler) onJuegatron(c tele.Context, member Member) error {

	member.State.SetJuegatron()
	err := h.MembersDB.Update(context.Background(), member)
	if err != nil {
		c.Send("Wops! Algo ha ido mal. Inténtalo de nuevo")
		return fmt.Errorf("Failed to update DB, %w", err)
	}

	return c.Send("Listo! Ahora estas en modo juegatron. Puedes buscar juegos diciéndome parte del nombre.\nPor ejemplo, puedes buscar el \"Virus\" diciendo \"vir\".", juegatronReplyMarkup())

}

func (h *Handler) OnExitJuegatron(c tele.Context) error {
	return h.IsAuthorized(h.onExitJuegatron)(c)
}

func (h *Handler) onExitJuegatron(c tele.Context, member Member) error {

	member.State.Clear()
	err := h.MembersDB.Update(context.Background(), member)
	if err != nil {
		c.Send("Wops! Algo ha ido mal. Inténtalo de nuevo")
		return fmt.Errorf("Failed to update DB, %w", err)
	}

	return c.Send("Vuelves a estar en modo normal", mainMenuReplyMarkup(member))

}

func (h *Handler) onJuegatronText(c tele.Context, member Member) error {
	ctx, cancel := GetContext(c)
	defer cancel()

	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "JuegatronText"), c.Sender())

	log = log.WithField(ilog.FieldText, c.Text())

	gameList, err := h.JuegatronGameDB.List(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to cache game data")
		return c.Send("Wops! Algo ha ido mal, vuelve a intentarlo en unos momentos. Si el problema persiste, Avisa a @MetalBlueberry")
	}

	list, err := h.textSearchGame(log, c, gameList, c.Text(), juegatronReplyMarkup())

	if len(list) == 0 {
		return nil
	}

	switch {
	case len(list) <= 3:
		for _, g := range list {
			log.WithField("Game", g.Name).Info("Found Game")
			err := c.Send(g.JuegatronCard(), g.JuegatronButtons())
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
			err := c.Send(block, juegatronReplyMarkup())
			if err != nil {
				log.Error(err)
			}
			if ctx.Err() != nil {
				return c.Send(fmt.Sprintf("Wops! Parece que no me ha dado tiempo... Envíame algo mas simple.\n%s", ctx.Err()))
			}
		}
	}
	return nil
}

func (h *Handler) OnJuegatronReturn(c tele.Context) error {
	return h.IsAuthorized(h.onJuegatronReturn)(c)
}

func (h *Handler) onJuegatronReturn(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "JuegatronReturn"), c.Sender())

	g, err := NewGameFromCard(c.Message().Text)
	if err != nil {
		c.Edit("Wops! Algo ha ido mal....\nInténtalo de nuevo")
		return fmt.Errorf("failed to load data form card, %w", err)
	}
	log = log.
		WithField("Game", g.Name).
		WithField("ID", g.ID)

	games, err := h.JuegatronGameDB.List(context.Background())
	if err != nil {
		log.WithError(err).Error("Unable to list from Juegatron GameDB")
		c.Edit(err.Error())
		return c.Respond()
	}
	getResult, err := Games(games).Get(g.ID, g.Name)
	if err != nil {
		log.WithError(err).Error("Unable to get from Juegatron GameDB")
		c.Edit(err.Error())
		return c.Respond()
	}
	if getResult == nil {
		log.Warn("Unable to find game")
		c.Edit("No he podido encontrar el juego. Intenta volver a buscarlo, tal vez se ha modificado el excel")
		return c.Respond()
	}

	g = *getResult

	if g.IsAvailable() {
		err := c.Edit("Parece que alguien ha modificado los datos. te envío los últimos actualizados")
		if err != nil {
			log.Print(err)
		}
		err = c.Send(g.JuegatronCard(), g.JuegatronButtons())
		if err != nil {
			log.Print(err)
		}
		log.Info("Conflict on Return")
		return c.Respond()
	}

	if g.IsAvailable() {
		c.Edit(g.JuegatronCard(), g.JuegatronButtons())
		return c.Send("Parece que alguien ha devuelvo ya este juego...")
	}

	previousHolder := g.Holder

	g.Return()

	h.JuegatronAudit.AuditDB.Append(context.Background(), []JuegatronAuditEntry{
		NewJuegatronAuditEntry(g, member),
	})

	c.Edit(g.JuegatronCard(), g.JuegatronButtons())
	log.Info("Game returned")

	selector := &tele.ReplyMarkup{}
	rows := []tele.Row{}

	rows = append(rows, selector.Row(
		selector.Data("Deshacer", "undo-juegatron-return", g.LineData()),
	))

	selector.Inline(rows...)
	c.Send(fmt.Sprintf("Devuelto %s %s\nLo tenia....", g.ID, g.Name), selector)
	c.Send(previousHolder)
	return c.Respond()
}

func (h *Handler) OnUndoJuegatronReturn(c tele.Context) error {
	return h.IsAuthorized(h.onUndoJuegatronReturn)(c)
}

func (h *Handler) onUndoJuegatronReturn(c tele.Context, member Member) error {
	defer c.Respond()

	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "UndoJuegatronReturn"), c.Sender())
	g := NewGameFromLineData(c.Data())
	log = log.
		WithField("Game", g.Name).
		WithField("ID", g.ID)

	entries, err := h.JuegatronAudit.AuditDB.List(context.Background())
	if err != nil {
		log.WithError(err).Error("Failed to list entries")
		return c.Send("Wops! Algo ha ido mal, vuelve a intentarlo")

	}
	var lastEntry *JuegatronAuditEntry
	for i := range entries {
		if entries[i].ID == g.ID && entries[i].Actor == member.Nickname {
			lastEntry = &entries[i]
		}
	}
	if lastEntry == nil {
		return c.Send("Wops! No he podido encontrar tu ultima acción. Mejor revisa el excel a mano por si acaso")
	}
	err = h.JuegatronAudit.AuditDB.Delete(context.Background(), *lastEntry)
	if err != nil {
		log.WithError(err).Error("Failed to delete entries")
		return c.Send("Wops! Algo ha ido mal, vuelve a intentarlo")
	}
	c.Edit("Okey! Hemos vuelto atrás en el tiempo")

	games, err := h.JuegatronGameDB.List(context.Background())
	if err != nil {
		log.WithError(err).Error("Failed to list games")
		return nil
	}
	getGame, _ := Games(games).Get(g.ID, g.Name)
	return c.Send(getGame.JuegatronCard(), getGame.JuegatronButtons())
}

func (h *Handler) OnJuegatronTake(c tele.Context) error {
	return h.IsAuthorized(h.onJuegatronTake)(c)
}

func (h *Handler) onJuegatronTake(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "JuegatronTake"), c.Sender())
	defer c.Respond()

	g, err := NewGameFromCard(c.Message().Text)
	if err != nil {
		c.Edit("Wops! Algo ha ido mal....\nInténtalo de nuevo")
		return fmt.Errorf("failed to load data form card, %w", err)
	}

	log = log.
		WithField("Game", g.Name).
		WithField("ID", g.ID)

	games, err := h.JuegatronGameDB.List(context.Background())
	if err != nil {
		log.WithError(err).Error("Unable to list from Juegatron GameDB")
		c.Edit(err.Error())
		return c.Respond()
	}
	getResult, err := Games(games).Get(g.ID, g.Name)
	if err != nil {
		log.WithError(err).Error("Unable to get from Juegatron GameDB")
		c.Edit(err.Error())
		return c.Respond()
	}
	if err != nil {
		log.WithError(err).Error("Unable to get game from DB")
		return c.Edit(err.Error())
	}
	if getResult == nil {
		log.Warn("Unable to find game")
		return c.Edit("No he podido encontrar el juego. Intenta volver a buscarlo, tal vez se ha modificado el excel")
	}

	g = *getResult

	if !g.IsAvailable() {
		err := c.Edit("Parece que alguien ha modificado los datos, te envío los últimos actualizados")
		if err != nil {
			log.Error(err)
		}
		err = c.Send(g.JuegatronCard(), g.JuegatronButtons())
		if err != nil {
			log.Error(err)
		}
		log.Info("Conflict on Take")
		return err
	}

	member.State.SetJuegatronWaitingForName(g)

	err = h.MembersDB.Update(context.Background(), member)
	if err != nil {
		log.WithError(err).Error("failed to update memberDB")
		return c.Send("Algo ha ido mal, vuelve a intentarlo")
	}
	return c.Send("Dime el nombre de la persona o el DNI.", cancelJuegatronMenu)
}

func (h *Handler) onJuegatronTakeWaitForName(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "JuegatronTakeWaitForName"), c.Sender())

	g := NewGameFromLineData(member.State.Data)
	log = log.
		WithField("Game", g.Name).
		WithField("ID", g.ID)

	member.State.SetJuegatron()
	err := h.MembersDB.Update(context.Background(), member)
	if err != nil {
		log.WithError(err).Warn("Failed to update member DB")
	}

	games, err := h.JuegatronGameDB.List(context.Background())
	if err != nil {
		log.WithError(err).Error("Unable to list from Juegatron GameDB")
		c.Edit(err.Error())
		return c.Respond()
	}
	getResult, err := Games(games).Get(g.ID, g.Name)
	if err != nil {
		log.WithError(err).Error("Unable to get from Juegatron GameDB")
		c.Edit(err.Error())
		return c.Respond()
	}
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

	if !g.IsAvailable() {
		c.Edit(g.JuegatronCard(), g.JuegatronButtons())
		log.Info("Conflict on take")
		return c.Send("El juego no está disponible")
	}

	g.Take(c.Text())

	h.JuegatronAudit.AuditDB.Append(context.Background(), []JuegatronAuditEntry{
		NewJuegatronAuditEntry(g, member),
	})

	c.Edit(g.JuegatronCard(), g.JuegatronButtons())
	log.Info("Game taken")

	c.Send(fmt.Sprintf("Listo! has dado el juego a %s", c.Text()), juegatronReplyMarkup())
	c.Send(g.JuegatronCard(), g.JuegatronButtons())
	return c.Respond()
}

func (h *Handler) OnCancelJuegatron(c tele.Context) error {
	return h.IsAuthorized(h.onCancelJuegatron)(c)
}
func (h *Handler) onCancelJuegatron(c tele.Context, member Member) error {
	log := ilog.WithTelegramUser(logrus.WithField(ilog.FieldHandler, "CancelJuegatron"), c.Sender())
	member.State.SetJuegatron()

	err := h.MembersDB.Update(context.Background(), member)
	if err != nil {
		log.WithError(err).Warn("Failed to update member DB")
		return c.Send("Wops! Algo ha ido mal, inténtalo de nuevo")
	}

	log.Info("done")
	return c.Send("Okey, Cancelo el préstamo", juegatronReplyMarkup())
}

func (h *Handler) OnListJuegatron(c tele.Context) error {
	return h.IsAuthorized(h.onListJuegatron)(c)
}

func (h *Handler) onListJuegatron(c tele.Context, member Member) error {

	log := ilog.WithTelegramUser(logrus.
		WithField(ilog.FieldHandler, "listJuegatron"),
		c.Sender())

	gameList, err := h.JuegatronGameDB.List(context.TODO())
	if err != nil {
		return c.Send(err.Error())
	}

	if len(gameList) == 0 {
		return c.Send("No se han encontrado juegos")
	}

	for _, block := range SendList(gameList) {
		err := c.Send(block, juegatronReplyMarkup())
		if err != nil {
			log.Error(err)
		}
	}

	return nil
}
