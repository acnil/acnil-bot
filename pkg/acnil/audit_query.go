package acnil

import (
	"context"
	"fmt"
	"time"
)

type AuditDatabaseRO interface {
	List(ctx context.Context) ([]AuditEntry, error)
}

type AuditQuery struct {
	AuditDB AuditDatabase
}

type Query struct {
	From, To time.Time
	Game     *Game
	Limit    int
	Member   *Member
}

func (a *AuditQuery) Find(ctx context.Context, query Query) ([]AuditEntry, error) {
	entries, err := a.AuditDB.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to list audit entries, %w", err)
	}

	result := []AuditEntry{}
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]

		if !query.From.IsZero() && e.Timestamp.Before(query.From) {
			continue
		}

		if !query.To.IsZero() && e.Timestamp.After(query.To) {
			continue
		}

		if query.Game != nil && !e.Game().IsTheSameGame(*query.Game) {
			continue
		}

		if query.Member != nil && !e.Game().IsHeldBy(*query.Member) {
			continue
		}

		result = append(result, e)
		if query.Limit != 0 && len(result) == query.Limit {
			break
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	reverse(result)
	return result, nil
}
