package views

import (
	"cosoft-cli/internal/ui/slack"
	"encoding/json"
	"fmt"
)

type Action struct {
	ActionID string          `json:"action_id"`
	Values   json.RawMessage `json:"values"`
}

type View interface {
	Update(action Action) (View, Cmd)
}

type Cmd any

func RestoreView(messageType string, payload []byte) (View, error) {
	var view View

	switch messageType {
	case "login":
		view = &LoginView{}
	case "landing":
		view = &LandingView{}
	case "quick-book":
		view = &QuickBookView{}
	case "browse":
		view = &BrowseView{}
	case "reservations":
		view = &ReservationView{}
	default:
		return nil, fmt.Errorf("unknown view type: %s", messageType)
	}

	err := json.Unmarshal(payload, view)

	if err != nil {
		return nil, err
	}

	return view, nil
}

func ViewType(v View) string {
	switch v.(type) {
	case *LoginView:
		return "login"
	case *LandingView:
		return "landing"
	case *QuickBookView:
		return "quick-book"
	case *BrowseView:
		return "browse"
	case *ReservationView:
		return "reservations"
	default:
		return "unknown"
	}
}

func RenderView(v View) slack.Block {
	switch v := v.(type) {
	case *LoginView:
		return RenderLoginView(v)
	case *LandingView:
		return RenderLandingView(v)
	case *QuickBookView:
		return RenderQuickBookView(v)
	case *BrowseView:
		return RenderBrowseView(v)
	case *ReservationView:
		return RenderReservationsView(v)
	default:
		return slack.Block{}
	}
}
