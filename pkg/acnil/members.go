package acnil

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

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
	Row         string
	Nickname    string
	TelegramID  string
	Permissions MemberPermissions
}

func (m *Member) TelegramIDInt() int64 {
	i, _ := strconv.Atoi(m.TelegramID)
	return int64(i)
}

func (m *Member) Recipient() string {
	return m.TelegramID
}

func (m *Member) ToRow() (string, []interface{}) {
	return m.Row, []interface{}{
		m.Nickname,
		m.TelegramID,
		m.Permissions,
	}
}

func NewMembersDatabase(srv *sheets.Service, sheetID string) *SheetMembersDatabase {
	return &SheetMembersDatabase{
		SRV:       srv,
		ReadRange: "A:C",
		Sheet:     "Miembros",
		SheetID:   sheetID,
	}
}

func NewMemberFromTelegram(user *tele.User) Member {
	nickname := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	if user.Username == "" {
		nickname = user.Username
	}

	return Member{
		Row:         "",
		Nickname:    nickname,
		TelegramID:  strconv.Itoa(int(user.ID)),
		Permissions: PermissionNo,
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
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	members := []Member{}

	if len(resp.Values) == 0 {
		return members, nil
	}

	for i, row := range resp.Values[1:] {
		if len(row) < MemberColumns {
			continue
		}
		members = append(members, NewMemberFromRow(db.rowReadRange(i+2), row))
	}
	return members, nil
}

func (db *SheetMembersDatabase) Append(ctx context.Context, member Member) error {
	values := []interface{}{}

	values = append(values, member.Nickname, member.TelegramID, member.Permissions)

	_, err := db.SRV.Spreadsheets.Values.Append(db.SheetID, db.fullReadRange(), &sheets.ValueRange{Values: [][]interface{}{values}}).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Fatalf("Unable to append data to sheet: %v", err)
	}
	return nil
}

func (db *SheetMembersDatabase) Update(ctx context.Context, member Member) error {
	range_, row := member.ToRow()

	request := db.SRV.Spreadsheets.Values.Update(db.SheetID, member.Row, &sheets.ValueRange{
		Range: range_,
		Values: [][]interface{}{
			row,
		},
	})
	request.ValueInputOption("USER_ENTERED")
	_, err := request.Do()
	if err != nil {
		return err
	}
	return nil
}

const (
	MemberColumns          = 3
	MemberColumnNickname   = 0
	MemberColumnTelegramID = 1
	MemberColumnPermission = 2
)

func NewMemberFromRow(range_ string, row []interface{}) Member {
	fullrow := make([]string, MemberColumns)
	for i := range fullrow {
		fullrow[i] = ""
	}
	for i := range row {
		fullrow[i] = row[i].(string)
	}
	return Member{
		Row:         range_,
		Nickname:    fullrow[MemberColumnNickname],
		TelegramID:  fullrow[MemberColumnTelegramID],
		Permissions: ParseMemberPermissions(fullrow[MemberColumnPermission]),
	}
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
