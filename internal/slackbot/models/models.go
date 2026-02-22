package models

import (
	"cosoft-cli/internal/storage"
	"encoding/json"
)

const (
	errInternalError = ":red_circle: Erreur interne, veuillez vous reconnecter"
)

// State represents the state of the application.
type State interface {
	Update(*storage.Store, UpdateParams) (State, error)
}

// UpdateParams stores required parameters for [State.Update].
type UpdateParams struct {
	UserID      string
	MessageType string
	ActionID    string
	Values      json.RawMessage
}

// ptr returns a pointer to a value.
//
// TODO: should be replaced by new in Go v1.26.
func ptr[T any](t T) *T { return &t }
