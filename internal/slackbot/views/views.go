package views

import (
	"cosoft-cli/internal/slackbot/models"
	"cosoft-cli/internal/ui/slack"
)

func RenderState(s models.State) slack.Block {
	switch s := s.(type) {
	case *models.LoginState:
		return RenderLoginView(s)
	case *models.LandingState:
		return RenderLandingView(s)
	case *models.QuickBookState:
		return RenderQuickBookView(s)
	case *models.BrowseState:
		return RenderBrowseView(s)
	case *models.ReservationState:
		return RenderReservationsView(s)
	default:
		return slack.Block{}
	}
}
