package views

import (
	"cosoft-cli/internal/slackbot/models"
	"cosoft-cli/internal/ui/slack"
)

func RenderLandingView(s *models.LandingState) slack.Block {
	return slack.MainMenu(*s.User)
}
