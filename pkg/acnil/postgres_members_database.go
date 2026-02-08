package acnil

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type PostgresMembersDatabase struct {
	db *sql.DB
}

func NewPostgresMembersDatabase(db *sql.DB) *PostgresMembersDatabase {
	return &PostgresMembersDatabase{db: db}
}

func (db *PostgresMembersDatabase) Get(ctx context.Context, telegramID int64) (*Member, error) {
	query := `
		SELECT id, nickname, telegram_id, permissions, state_action, state_data,
			   telegram_name, telegram_username
		FROM members
		WHERE telegram_id = $1
	`

	var member Member
	var stateAction, stateData sql.NullString

	err := db.db.QueryRowContext(ctx, query, strconv.Itoa(int(telegramID))).Scan(
		&member.Row, // Using id as row for compatibility
		&member.Nickname,
		&member.TelegramID,
		&member.Permissions,
		&stateAction,
		&stateData,
		&member.TelegramName,
		&member.TelegramUsername,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	// Reconstruct MemberState from separate fields
	if stateAction.Valid && stateAction.String != "" {
		member.State = MemberState{
			Action: StateAction(stateAction.String),
		}
		if stateData.Valid {
			member.State.Data = stateData.String
		}
	}

	return &member, nil
}

func (db *PostgresMembersDatabase) List(ctx context.Context) ([]Member, error) {
	query := `
		SELECT id, nickname, telegram_id, permissions, state_action, state_data,
			   telegram_name, telegram_username
		FROM members
		ORDER BY nickname
	`

	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query members: %w", err)
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var member Member
		var stateAction, stateData sql.NullString

		err := rows.Scan(
			&member.Row, // Using id as row for compatibility
			&member.Nickname,
			&member.TelegramID,
			&member.Permissions,
			&stateAction,
			&stateData,
			&member.TelegramName,
			&member.TelegramUsername,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member row: %w", err)
		}

		// Reconstruct MemberState from separate fields
		if stateAction.Valid && stateAction.String != "" {
			member.State = MemberState{
				Action: StateAction(stateAction.String),
			}
			if stateData.Valid {
				member.State.Data = stateData.String
			}
		}

		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating member rows: %w", err)
	}

	return members, nil
}

func (db *PostgresMembersDatabase) Append(ctx context.Context, member Member) error {
	query := `
		INSERT INTO members (nickname, telegram_id, permissions, state_action, state_data,
							telegram_name, telegram_username)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var stateAction, stateData sql.NullString
	if member.State.Action != "" {
		stateAction = sql.NullString{String: string(member.State.Action), Valid: true}
	}
	if member.State.Data != "" {
		stateData = sql.NullString{String: member.State.Data, Valid: true}
	}

	_, err := db.db.ExecContext(ctx, query,
		member.Nickname,
		member.TelegramID,
		member.Permissions,
		stateAction,
		stateData,
		member.TelegramName,
		member.TelegramUsername,
	)

	if err != nil {
		return fmt.Errorf("failed to append member: %w", err)
	}

	logrus.WithField("nickname", member.Nickname).Info("Successfully appended member to PostgreSQL")
	return nil
}

func (db *PostgresMembersDatabase) Update(ctx context.Context, member Member) error {
	query := `
		UPDATE members
		SET nickname = $1, permissions = $2, state_action = $3, state_data = $4,
			telegram_name = $5, telegram_username = $6, updated_at = NOW()
		WHERE telegram_id = $7
	`

	var stateAction, stateData sql.NullString
	if member.State.Action != "" {
		stateAction = sql.NullString{String: string(member.State.Action), Valid: true}
	}
	if member.State.Data != "" {
		stateData = sql.NullString{String: member.State.Data, Valid: true}
	}

	result, err := db.db.ExecContext(ctx, query,
		member.Nickname,
		member.Permissions,
		stateAction,
		stateData,
		member.TelegramName,
		member.TelegramUsername,
		member.TelegramID,
	)

	if err != nil {
		return fmt.Errorf("failed to update member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no member found with telegram_id: %s", member.TelegramID)
	}

	logrus.WithField("nickname", member.Nickname).Info("Successfully updated member in PostgreSQL")
	return nil
}
