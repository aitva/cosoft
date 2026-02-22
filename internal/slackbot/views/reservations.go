package views

import (
	"cosoft-cli/internal/slackbot/models"
	"cosoft-cli/internal/ui/slack"
	"fmt"
	"time"
)

func RenderReservationsView(s *models.ReservationState) slack.Block {
	if s.Error != nil {
		return slack.Block{
			Blocks: []slack.BlockElement{
				slack.NewContext(*s.Error),
			},
		}
	}

	switch s.Phase {
	case 0:
		blocks := slack.Block{
			Blocks: []slack.BlockElement{
				slack.NewHeader("Mes réservations"),
				slack.NewMrkDwn(fmt.Sprintf("*%d* réservation(s) à venir", len(*s.Reservations))),
			},
		}

		var list []slack.BlockElement

		for _, r := range *s.Reservations {
			parsedStart, err := time.Parse("2006-01-02T15:04:05", r.Start)

			if err != nil {
				fmt.Println(err)
				return slack.Block{}
			}

			parsedEnd, err := time.Parse("2006-01-02T15:04:05", r.End)

			if err != nil {
				fmt.Println(err)
				return slack.Block{}
			}

			duration := parsedEnd.Sub(parsedStart).Minutes()

			paidPrice := r.Credits * (float64(duration) / 60)

			dateFormat := "02/01/2006 15:04"
			list = append(list, slack.BlockElement(slack.NewMenuItem(
				fmt.Sprintf(
					"*%s*\n%s → %s · %.02f crédits",
					r.ItemName,
					parsedStart.Format(dateFormat),
					parsedEnd.Format(dateFormat),
					paidPrice,
				),
				"Sélectionner",
				r.OrderResourceRentId,
			)))
		}

		if s.PickedReservation != nil {

			if s.BookingStarted {
				list = append(
					list,
					slack.BlockElement(slack.NewContext(":warning: Impossible d'annuler une réservation déjà commencée")),
				)
			} else {
				list = append(
					list,
					slack.BlockElement(slack.NewDivider()),
					slack.BlockElement(slack.NewMenuItem(
						fmt.Sprintf(
							"Confirmer l'annulation de \"%s\" ?",
							s.PickedReservation.ItemName,
						),
						"Annuler réservation",
						"cancel",
					)),
				)
			}
		}

		list = append(
			list,
			slack.BlockElement(slack.NewDivider()),
			slack.NewButtons([]slack.ChoicePayload{{"Retour", "back"}}),
		)

		blocks.Blocks = list

		return blocks
	case 1:
		return slack.Block{
			Blocks: []slack.BlockElement{
				slack.NewHeader(":white_check_mark: Annulation réussie réussie !"),
				slack.NewMrkDwn("Vous pouvez maintenant retourner à l'accueil"),
				slack.NewDivider(),
				slack.NewButtons([]slack.ChoicePayload{{"Retour", "back"}}),
			},
		}
	default:
		return slack.Block{}
	}

}
