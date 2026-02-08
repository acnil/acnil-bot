package acnil

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type PostgresAuditDatabase struct {
	db *sql.DB
}

func NewPostgresAuditDatabase(db *sql.DB) *PostgresAuditDatabase {
	return &PostgresAuditDatabase{db: db}
}

func (db *PostgresAuditDatabase) Append(ctx context.Context, entries []AuditEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Begin transaction for consistency
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare the insert statement
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO audit_entries (timestamp, type, game_id, name, location, holder,
			comments, take_date, return_date, price, publisher, bgg)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert each entry
	for _, entry := range entries {
		// Set timestamp to current time if not already set
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}

		_, err := stmt.ExecContext(ctx,
			entry.Timestamp,
			string(entry.Type),
			entry.ID,
			entry.Name,
			entry.Location,
			entry.Holder,
			entry.Comments,
			entry.TakeDate,
			entry.ReturnDate,
			entry.Price,
			entry.Publisher,
			entry.BGG,
		)
		if err != nil {
			return fmt.Errorf("failed to insert audit entry: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logrus.WithField("count", len(entries)).Info("Successfully appended audit entries to PostgreSQL")
	return nil
}

func (db *PostgresAuditDatabase) List(ctx context.Context) ([]AuditEntry, error) {
	query := `
		SELECT timestamp, type, game_id, name, location, holder, comments,
			   take_date, return_date, price, publisher, bgg
		FROM audit_entries
		ORDER BY timestamp ASC
	`

	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit entries: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		var takeDate, returnDate sql.NullTime

		err := rows.Scan(
			&entry.Timestamp,
			&entry.Type,
			&entry.ID,
			&entry.Name,
			&entry.Location,
			&entry.Holder,
			&entry.Comments,
			&takeDate,
			&returnDate,
			&entry.Price,
			&entry.Publisher,
			&entry.BGG,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit entry row: %w", err)
		}

		if takeDate.Valid {
			entry.TakeDate = takeDate.Time
		}
		if returnDate.Valid {
			entry.ReturnDate = returnDate.Time
		}

		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit entry rows: %w", err)
	}

	return entries, nil
}
