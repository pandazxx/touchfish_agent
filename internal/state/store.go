package state

import "context"

// StateStore persists per-team state across restarts.
type StateStore interface {
	Load(ctx context.Context, teamName string) (*TeamState, error)
	Save(ctx context.Context, state *TeamState) error
	Delete(ctx context.Context, teamName string) error
}
