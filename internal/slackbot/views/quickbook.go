package views

import (
	"cosoft-cli/internal/common"
	"cosoft-cli/internal/slackbot/models"
	"cosoft-cli/internal/ui/slack"
	"fmt"
	"slices"
	"strconv"
	"time"
)

func RenderQuickBookView(s *models.QuickBookState) slack.Block {
	blocks := slack.QuickBookMenu()

	switch s.Phase {
	case 0:
		if s.Error != nil {
			blocks.Blocks = slices.Insert(
				blocks.Blocks,
				3,
				slack.BlockElement(slack.NewContext(*s.Error)),
			)
		}

		return blocks

	case 2:
		// Remove action buttons
		blocks.Blocks = blocks.Blocks[:len(blocks.Blocks)-1]
		// Add rest of the feedback
		blocks.Blocks = slices.Insert(
			blocks.Blocks,
			len(blocks.Blocks),
			slack.BlockElement(slack.NewMrkDwn(":large_green_circle: Une salle a été trouvée !")),
			slack.BlockElement(slack.NewMrkDwn("Réservation en cours...")),
		)

		return blocks
	case 3:
		duration, _ := strconv.Atoi(s.Duration)
		startTime := common.GetClosestQuarterHour()
		endTime := startTime.Add(time.Duration(duration) * time.Minute)
		dateFormat := "02/01/2006 15:04"
		paidPrice := s.PickedRoom.Price * (float64(duration) / 60)

		// Remove action buttons
		blocks.Blocks = blocks.Blocks[:len(blocks.Blocks)-1]
		// Add rest of the feedback
		blocks.Blocks = slices.Insert(
			blocks.Blocks,
			len(blocks.Blocks),
			slack.BlockElement(slack.NewDivider()),
			slack.BlockElement(slack.NewMrkDwn(":white_check_mark: *Réservation réussie !*")),

			slack.BlockElement(slack.NewMultiMarkdown([]string{
				fmt.Sprintf("*Salle de réunion :*\n%s", s.PickedRoom.Name),
				fmt.Sprintf("*Durée :*\n%s → %s", startTime.Format(dateFormat), endTime.Format(dateFormat)),
				fmt.Sprintf("*Coût :*\n%.2f credits", paidPrice),
			})),
			slack.BlockElement(slack.NewMenuItem(
				"Vous pouvez maintenant revenir à l'accueil",
				"Retour",
				"cancel",
			)),
		)

		return blocks
	default:
		return slack.QuickBookMenu()
	}
}
