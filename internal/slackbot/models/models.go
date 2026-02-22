package models

import (
	"cosoft-cli/internal/storage"
	"encoding/json"
	"fmt"
)

// List errors used in the package.
//
// TODO: finish to move errors here.
const (
	errInternalError = ":red_circle: Erreur interne, veuillez vous reconnecter"
)

// List state types used in the package.
const (
	browseStateType      = "browse"
	landingStateType     = "landing"
	loginStateType       = "login"
	quickBookStateType   = "quick-book"
	reservationStateType = "reservations"
)

// State is the interface that wraps all application states.
type State interface {
	// Type returns the state type.
	//
	// It is an unique ID used to save and retrive states from the database.
	Type() string

	// Update updates the state and returns the next state.
	//
	// It always returns a valid State even when an error occurs.
	Update(*storage.Store, UpdateParams) (State, error)

	// Next returns true when multiple updates are available.
	//
	// It is usefull to iterate through multiple phases of a message, but
	// it might have unforseen consequences later.
	Next() bool
}

// UpdateParams stores required parameters for [State.Update].
type UpdateParams struct {
	UserID      string
	MessageType string
	ActionID    string
	Values      json.RawMessage
}

// LoadState loads state from the store.
func LoadState(store *storage.Store, userID string) (State, error) {
	state, err := store.GetSlackState(userID)
	if err != nil {
		return nil, fmt.Errorf("get slack state: %v", err)
	}

	var s State
	switch state.MessageType {
	case browseStateType:
		s = &BrowseState{}
	case landingStateType:
		s = &LandingState{}
	case loginStateType:
		s = &LoginState{}
	case quickBookStateType:
		s = &QuickBookState{}
	case reservationStateType:
		s = &ReservationState{}
	default:
		return nil, fmt.Errorf("unknown state: %s", state.MessageType)
	}

	err = json.Unmarshal(state.Payload, s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// SaveState saves state in the store.
func SaveState(store *storage.Store, userID string, state State) error {
	return store.SetSlackState(userID, state.Type(), state)
}

// ptr returns a pointer to a value.
//
// TODO: should be replaced by new in Go v1.26.
func ptr[T any](t T) *T { return &t }
