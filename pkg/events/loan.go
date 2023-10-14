package events

import "time"

type Loans []Loan

func (l Loans) Last() Loan {
	return l[len(l)-1]
}

// Loan represents a game taken by someone
type Loan struct {
	// Row represents the row definition on google sheets
	Row string

	ID       string    `col:"0"`
	FullName string    `col:"1"`
	GameID   string    `col:"2"`
	Returned bool      `col:"3"`
	Time     time.Time `col:"4"`
}
