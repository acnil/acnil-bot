package acnil

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type PostgresGameDatabase struct {
	db *sql.DB
}

func NewPostgresGameDatabase(db *sql.DB) *PostgresGameDatabase {
	return &PostgresGameDatabase{db: db}
}

func (db *PostgresGameDatabase) Get(ctx context.Context, id string, name string) (*Game, error) {
	games, err := db.List(ctx)
	if err != nil {
		return nil, err
	}

	return Games(games).Get(id, name)
}

func (db *PostgresGameDatabase) List(ctx context.Context) ([]Game, error) {
	query := `
		SELECT id, name, location, holder, comments, take_date, return_date,
			   price, publisher, bgg, avg_rate, avg_weight, age, min_players,
			   max_players, playingtime, yearpublished, language_dependence
		FROM games
		ORDER BY id
	`

	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var game Game
		var takeDate, returnDate sql.NullTime

		err := rows.Scan(
			&game.ID,
			&game.Name,
			&game.Location,
			&game.Holder,
			&game.Comments,
			&takeDate,
			&returnDate,
			&game.Price,
			&game.Publisher,
			&game.BGG,
			&game.AvgRate,
			&game.AvgWeight,
			&game.Age,
			&game.MinPlayers,
			&game.MaxPlayers,
			&game.Playingtime,
			&game.Yearpublished,
			&game.LanguageDependence,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game row: %w", err)
		}

		if takeDate.Valid {
			game.TakeDate = takeDate.Time
		}
		if returnDate.Valid {
			game.ReturnDate = returnDate.Time
		}

		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating game rows: %w", err)
	}

	return games, nil
}

func (db *PostgresGameDatabase) Find(ctx context.Context, name string) ([]Game, error) {
	query := `
		SELECT id, name, location, holder, comments, take_date, return_date,
			   price, publisher, bgg, avg_rate, avg_weight, age, min_players,
			   max_players, playingtime, yearpublished, language_dependence
		FROM games
		WHERE LOWER(name) LIKE LOWER($1)
		ORDER BY id
	`

	rows, err := db.db.QueryContext(ctx, query, "%"+name+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search games: %w", err)
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var game Game
		var takeDate, returnDate sql.NullTime

		err := rows.Scan(
			&game.ID,
			&game.Name,
			&game.Location,
			&game.Holder,
			&game.Comments,
			&takeDate,
			&returnDate,
			&game.Price,
			&game.Publisher,
			&game.BGG,
			&game.AvgRate,
			&game.AvgWeight,
			&game.Age,
			&game.MinPlayers,
			&game.MaxPlayers,
			&game.Playingtime,
			&game.Yearpublished,
			&game.LanguageDependence,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game row: %w", err)
		}

		if takeDate.Valid {
			game.TakeDate = takeDate.Time
		}
		if returnDate.Valid {
			game.ReturnDate = returnDate.Time
		}

		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating game rows: %w", err)
	}

	return games, nil
}

func (db *PostgresGameDatabase) Update(ctx context.Context, games ...Game) error {
	if len(games) == 0 {
		return nil
	}

	// Build the query for batch update
	placeholders := make([]string, len(games))
	args := make([]interface{}, 0, len(games)*19) // 19 fields per game

	for i, game := range games {
		placeholders[i] = fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*19+1, i*19+2, i*19+3, i*19+4, i*19+5, i*19+6, i*19+7, i*19+8, i*19+9,
			i*19+10, i*19+11, i*19+12, i*19+13, i*19+14, i*19+15, i*19+16, i*19+17,
			i*19+18, i*19+19,
		)

		args = append(args,
			game.ID, game.Name, game.Location, game.Holder, game.Comments,
			game.TakeDate, game.ReturnDate, game.Price, game.Publisher, game.BGG,
			game.AvgRate, game.AvgWeight, game.Age, game.MinPlayers, game.MaxPlayers,
			game.Playingtime, game.Yearpublished, game.LanguageDependence,
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO games (id, name, location, holder, comments, take_date, return_date,
						  price, publisher, bgg, avg_rate, avg_weight, age, min_players,
						  max_players, playingtime, yearpublished, language_dependence)
		VALUES %s
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			location = EXCLUDED.location,
			holder = EXCLUDED.holder,
			comments = EXCLUDED.comments,
			take_date = EXCLUDED.take_date,
			return_date = EXCLUDED.return_date,
			price = EXCLUDED.price,
			publisher = EXCLUDED.publisher,
			bgg = EXCLUDED.bgg,
			avg_rate = EXCLUDED.avg_rate,
			avg_weight = EXCLUDED.avg_weight,
			age = EXCLUDED.age,
			min_players = EXCLUDED.min_players,
			max_players = EXCLUDED.max_players,
			playingtime = EXCLUDED.playingtime,
			yearpublished = EXCLUDED.yearpublished,
			language_dependence = EXCLUDED.language_dependence,
			updated_at = NOW()
	`, strings.Join(placeholders, ", "))

	_, err := db.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update games: %w", err)
	}

	logrus.WithField("count", len(games)).Info("Successfully updated games in PostgreSQL")
	return nil
}
