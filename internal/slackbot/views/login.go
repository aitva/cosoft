package views

import (
	"cosoft-cli/internal/slackbot/models"
	"cosoft-cli/internal/ui/slack"
	"slices"
)

func RenderLoginView(s *models.LoginState) slack.Block {
	loginBlocks := []slack.BlockElement{
		slack.NewMrkDwn(":information_source:  Pour réserver une salle, il faut d'abord vous identifier."),
		slack.NewInput("Email", "email"),
		slack.NewInput("Mot de passe", "password"),
		slack.NewContext(":warning: Le mot de passe est affiché en clair dans le champ"),
		slack.NewButtons([]slack.ChoicePayload{{"Connexion", "login"}}),
	}

	if s.Error != nil {
		loginBlocks = slices.Insert(
			loginBlocks,
			3,
			slack.BlockElement(slack.NewContext(*s.Error)),
		)
	}

	blocks := slack.Block{
		ResponseType: "ephemeral",
		Blocks:       loginBlocks,
	}

	return blocks
}
