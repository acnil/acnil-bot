package acnil

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/metalblueberry/acnil-bot/pkg/sheetsparser"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/sheets/v4"
	tele "gopkg.in/telebot.v3"
)

type SheetMembersDatabase struct {
	SRV       *sheets.Service
	ReadRange string
	Sheet     string
	SheetID   string
}

type MemberPermissions string

const (
	PermissionNo  MemberPermissions = "no"
	PermissionYes MemberPermissions = "si"
	// PermissionAdmin will be notified of new user logins so they can approve them from the app
	// This should not be used to restrict admin actions as it can be modified manually by any user
	PermissionAdmin MemberPermissions = "admin"
)

func (p MemberPermissions) IsAuthorised() bool {
	switch p {
	case PermissionYes, PermissionAdmin:
		return true
	default:
		return false
	}
}

type Member struct {
	Row string

	// Nickname Is the name used in the excel file and to set the Holder field on Games
	Nickname         string            `col:"0"`
	TelegramID       string            `col:"1"`
	Permissions      MemberPermissions `col:"2"`
	State            MemberState       `col:"3"`
	TelegramName     string            `col:"4"`
	TelegramUsername string            `col:"5"`
}

const (
	MemberColumns = 3
)

func (m *Member) TelegramIDInt() int64 {
	i, _ := strconv.Atoi(m.TelegramID)
	return int64(i)
}

func (m *Member) Recipient() string {
	return m.TelegramID
}

func NewMembersDatabase(srv *sheets.Service, sheetID string) *SheetMembersDatabase {
	return &SheetMembersDatabase{
		SRV:       srv,
		ReadRange: "A:F",
		Sheet:     "Miembros Telegram",
		SheetID:   sheetID,
	}
}

func NewMemberFromTelegram(user *tele.User) Member {
	name := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	nickname := name
	if strings.TrimSpace(user.Username) != "" {
		nickname = user.Username
	}

	return Member{
		Row:              "",
		Nickname:         strings.TrimSpace(nickname),
		TelegramID:       strconv.Itoa(int(user.ID)),
		Permissions:      PermissionNo,
		State:            MemberState{},
		TelegramName:     name,
		TelegramUsername: strings.TrimSpace(user.Username),
	}
}

func (db *SheetMembersDatabase) fullReadRange() string {
	return fmt.Sprintf("%s!%s", db.Sheet, db.ReadRange)
}
func (db *SheetMembersDatabase) rowReadRange(row int) string {
	return fmt.Sprintf("%s!%d:%d", db.Sheet, row, row)
}

func (db *SheetMembersDatabase) Get(ctx context.Context, telegramID int64) (*Member, error) {
	members, err := db.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		if member.TelegramID == strconv.Itoa(int(telegramID)) {
			return &member, nil
		}
	}
	return nil, nil
}

func (db *SheetMembersDatabase) List(ctx context.Context) ([]Member, error) {
	resp, err := db.SRV.Spreadsheets.Values.Get(db.SheetID, db.fullReadRange()).Do()
	if err != nil {
		logrus.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	members := []Member{}

	if len(resp.Values) == 0 {
		return members, nil
	}

	for i, row := range resp.Values[1:] {
		if len(row) < MemberColumns {
			continue
		}
		m := Member{
			Row: db.rowReadRange(i + 2),
		}
		sheetsparser.Unmarshal(row, &m)

		members = append(members, m)
	}
	return members, nil
}

func (db *SheetMembersDatabase) Append(ctx context.Context, member Member) error {
	values, err := sheetsparser.Marshal(&member)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal member, %w", err)
	}

	_, err = db.SRV.Spreadsheets.Values.Append(db.SheetID, db.fullReadRange(), &sheets.ValueRange{Values: [][]interface{}{values}}).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		logrus.Fatalf("Unable to append data to sheet: %v", err)
	}
	return nil
}

func (db *SheetMembersDatabase) Update(ctx context.Context, member Member) error {
	values, err := sheetsparser.Marshal(&member)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal member, %w", err)
	}

	request := db.SRV.Spreadsheets.Values.Update(db.SheetID, member.Row, &sheets.ValueRange{
		Range: member.Row,
		Values: [][]interface{}{
			values,
		},
	})
	request.ValueInputOption("USER_ENTERED")
	_, err = request.Do()
	if err != nil {
		return err
	}
	return nil
}

func ParseMemberPermissions(p string) MemberPermissions {
	switch {
	case strings.EqualFold(p, string(PermissionYes)):
		return PermissionYes
	case strings.EqualFold(p, string(PermissionAdmin)):
		return PermissionAdmin
	default:
		return PermissionNo
	}
}

type StateAction string

const (
	StateActionRename                  StateAction = "rename"
	StateActionUpdateComment           StateAction = "update-comment"
	StateGetGamesTakenByUser           StateAction = "get-games-taken-by-user"
	StateActionJuegatron               StateAction = "juegatron"
	StateActionJuegatronWaitingForName StateAction = "juegatron-waiting-for-name"
)

type MemberState struct {
	Action StateAction `json:"action,omitempty"`
	Data   string      `json:"data,omitempty"`
}

func (s *MemberState) Clear() {
	s.Action = ""
	s.Data = ""
}

func (s *MemberState) Is(action StateAction) bool {
	return s.Action == action
}

func (s *MemberState) SetRename() {
	s.Action = StateActionRename
}

func (s *MemberState) SetUpdateComment(g Game) {
	s.Action = StateActionUpdateComment
	s.Data = g.LineData()
}

func (s *MemberState) SetGetGamesTakenByUser() {
	s.Action = StateGetGamesTakenByUser
}

func (s *MemberState) SetJuegatron() {
	s.Action = StateActionJuegatron
}

func (s *MemberState) SetJuegatronWaitingForName(g Game) {
	s.Action = StateActionJuegatronWaitingForName
	s.Data = g.LineData()
}
