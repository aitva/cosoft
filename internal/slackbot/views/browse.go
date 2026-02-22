package views

import (
	"cosoft-cli/internal/slackbot/models"
	"cosoft-cli/internal/ui/slack"
	"fmt"
	"slices"
	"strconv"
	"time"
)

func RenderBrowseView(s *models.BrowseState) slack.Block {
	switch s.Phase {
	case 0:
		blocks := slack.BrowseMenu()
		if s.Error != nil {
			blocks.Blocks = slices.Insert(
				blocks.Blocks,
				5,
				slack.BlockElement(slack.NewContext(*s.Error)),
			)
		}

		return blocks
	case 1:
		if len(*s.Rooms) == 0 {
			return slack.Block{
				Blocks: []slack.BlockElement{
					slack.NewMenuItem(
						"*Aucune salle disponible*\nVeuillez changer vos filtres",
						"Retour",
						"back",
					),
				},
			}
		}

		nbPeople, duration, _ := filtersToNumber(s)
		t, _ := criteriaToTime(s)
		end := t.Add(time.Duration(duration) * time.Minute)
		choices := make([]slack.ChoicePayload, len(*s.Rooms))

		for i, room := range *s.Rooms {
			choices[i] = slack.ChoicePayload{
				Text:  room.Name,
				Value: room.Id,
			}
		}

		blocks := []slack.BlockElement{
			slack.NewHeader(fmt.Sprintf("%d salles ont été trouvées", len(*s.Rooms))),
			slack.NewMrkDwn(fmt.Sprintf(
				"%s → %s — %d personnes",
				t.Format("02/01/2006 15:04"),
				end.Format("02/01/2006 15:04"),
				nbPeople,
			)),
			slack.NewDivider(),
			slack.NewSelect(
				"Sélectionez une salle",
				"Salle",
				"pick-room",
				choices,
			),
		}

		if s.PickedRoom != nil {
			blocks = append(
				blocks,
				slack.NewPreview(
					fmt.Sprintf("*%s*\n%.2f crédits", s.PickedRoom.Name, s.PickedRoom.Price),
					s.PickedRoom.Image,
					s.PickedRoom.Name,
				),
				slack.NewButtons([]slack.ChoicePayload{{"Réserver", "book"}}),
			)
		}

		blocks = append(blocks,
			slack.NewDivider(),
			slack.NewButtons([]slack.ChoicePayload{
				{"Retour à l'accueil", "cancel"},
				{"Modifier les filtres", "back"},
			}),
		)

		return slack.Block{
			Blocks: blocks,
		}

	case 2:
		duration, _ := strconv.Atoi(s.Duration)
		startTime, _ := criteriaToTime(s)
		endTime := startTime.Add(time.Duration(duration) * time.Minute)
		dateFormat := "02/01/2006 15:04"
		paidPrice := s.PickedRoom.Price * (float64(duration) / 60)

		return slack.Block{
			Blocks: []slack.BlockElement{
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
			},
		}
	}
	return slack.Block{}
}

func criteriaToTime(s *models.BrowseState) (*time.Time, error) {
	dt := fmt.Sprintf("%s %s", s.Date, s.Time)

	parsedDt, err := time.Parse("2006-01-02 15:04", dt)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &parsedDt, nil
}

func filtersToNumber(s *models.BrowseState) (int, int, error) {
	nbPeople, err := strconv.Atoi(s.NbPeople)

	if err != nil {
		fmt.Println(err)
		return 0, 0, err
	}

	duration, err := strconv.Atoi(s.Duration)

	if err != nil {
		fmt.Println(err)
		return 0, 0, err
	}

	return nbPeople, duration, nil
}
