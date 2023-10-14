package events

import (
	"context"
)

type GameDatabase interface {
	List(ctx context.Context) ([]Game, error)
	Append(ctx context.Context, game Game) error
	Delete(ctx context.Context, game Game) error
}

type LoanDatabase interface {
	List(ctx context.Context) (Loans, error)
	Append(ctx context.Context, loan Loan) error
	Update(ctx context.Context, loan Loan) error
}

type Event struct {
	GameDB GameDatabase
	LoanDB LoanDatabase
}

func New(GameDB GameDatabase, LoanDB LoanDatabase) *Event {
	return &Event{
		GameDB: GameDB,
		LoanDB: LoanDB,
	}
}

// ListGames returns a list of all the available games for loaning
func (e *Event) ListGames(ctx context.Context, game Game) ([]Game, error) {
	panic("Not Implemented Yet")
}

// GetGame returns a single game by ID
func (e *Event) GetGame(ctx context.Context, gameID string) (Game, error) {
	panic("Not Implemented Yet")
}

// AddGame Adds a game to the list of available games
func (e *Event) AddGame(ctx context.Context, game Game) error {
	panic("Not Implemented Yet")
}

// RemoveGame Removes a game from the list of available games
func (e *Event) RemoveGame(ctx context.Context, gameID string) error {
	panic("Not Implemented Yet")
}

// GetGameLoan returns the loan status for a given game
func (e *Event) GetGameLoan(ctx context.Context, gameID string) (Loan, error) {
	panic("Not Implemented Yet")
}

// LoanGame creates a new Loan entry for the game
func (e *Event) LoanGame(ctx context.Context, loan Loan) error {
	panic("Not Implemented Yet")
}

// ReturnGame updates the loan status to "returned"
func (e *Event) ReturnGame(ctx context.Context, gameID string) error {
	panic("Not Implemented Yet")
}
