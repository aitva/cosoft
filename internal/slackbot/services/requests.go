package services

import (
	"bytes"
	"cosoft-cli/internal/api"
	"cosoft-cli/internal/slackbot/views"
	"cosoft-cli/internal/storage"
	"cosoft-cli/internal/ui/slack"
	"cosoft-cli/shared/models"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (s *SlackService) HandleInteraction(payload string) error {
	var result models.InteractionDiscovery

	err := json.Unmarshal([]byte(payload), &result)
	if err != nil {
		return err
	}

	dbView, err := s.store.GetSlackState(result.User.ID)
	if err != nil {
		return err
	}

	var view views.View
	if dbView != nil {
		view, err = views.RestoreView(dbView.MessageType, dbView.Payload)
		if err != nil {
			return err
		}

		if view == nil {
			return fmt.Errorf("no active view for user %s", result.User.ID)
		}
	}

	newView, cmd := view.Update(views.Action{
		ActionID: result.Actions[0].ActionID,
		Values:   result.State.Values,
	})

	user, err := s.store.GetUserData(&result.User.ID)
	if err != nil {
		return err
	}

	switch c := cmd.(type) {
	case *views.LoginCmd:
		err = s.LogInUser(c.Email, c.Password, result.User.ID)

		if err != nil {
			errMsg := ":red_circle: Identifiant / mot de passe incorrect"
			if loginView, ok := newView.(*views.LoginView); ok {
				loginView.Error = &errMsg
			}
		} else {
			user, err := s.store.GetUserData(&result.User.ID)

			if err != nil {
				return err
			}

			newView = &views.LandingView{
				User: *user,
			}
		}
	case *views.LandingCmd:
		user, err := s.RefreshAndGetUser(result.User.ID)

		if err != nil {
			return err
		}

		newView = &views.LandingView{
			User: *user,
		}
	case *views.QuickBookCmd:
		rooms, err := s.getRoomAvailabilities(
			*user,
			c.NbPeople,
			c.Duration,
			c.Datetime,
		)

		if err != nil {
			errMsg := ":red_circle: La réservation a échoué"
			if qbView, ok := newView.(*views.QuickBookView); ok {
				qbView.Error = &errMsg
			}
		} else {
			qbView := newView.(*views.QuickBookView)
			qbView.Phase = 2
			qbView.Rooms = &rooms

			err := s.store.SetSlackState(result.User.ID, views.ViewType(qbView), qbView)

			if err != nil {
				return err
			}

			blocks := views.RenderView(qbView)
			err = s.SendToSlack(result.ResponseURL, blocks)

			if err != nil {
				return err
			}

			var pickedRoom *models.Room

			for _, room := range rooms {
				if room.NbUsers >= c.NbPeople {
					pickedRoom = &room
					break
				}
			}

			if pickedRoom == nil {
				return fmt.Errorf(":red_circle: Aucune salle disponible")
			}

			if user.Credits < pickedRoom.Price {
				return fmt.Errorf(":red_circle: Pas assez de crédits pour faire une réservation")
			}

			err = s.bookRoom(
				*user,
				c.NbPeople,
				c.Duration,
				*pickedRoom,
				c.Datetime,
			)

			if err != nil {
				errMsg := err.Error()
				qbView.Error = &errMsg
			} else {
				qbView.PickedRoom = pickedRoom
				qbView.Phase = 3
			}

			err = s.store.SetSlackState(result.User.ID, views.ViewType(qbView), qbView)

			if err != nil {
				return err
			}

			blocks = views.RenderView(qbView)
			return s.SendToSlack(result.ResponseURL, blocks)
		}

	case *views.BrowseCmd:
		rooms, err := s.getRoomAvailabilities(
			*user,
			c.NbPeople,
			c.Duration,
			c.Datetime,
		)

		if err != nil {
			errMsg := ":red_circle: La réservation a échoué"
			if bView, ok := newView.(*views.BrowseView); ok {
				bView.Error = &errMsg
			}
		} else {
			bView := newView.(*views.BrowseView)
			bView.Phase = 1
			bView.Rooms = &rooms

			err = s.store.SetSlackState(result.User.ID, views.ViewType(bView), bView)

			if err != nil {
				return err
			}

			blocks := views.RenderView(bView)
			return s.SendToSlack(result.ResponseURL, blocks)
		}

	case *views.BookCmd:
		err = s.bookRoom(
			*user,
			c.NbPeople,
			c.Duration,
			c.PickedRoom,
			c.Datetime,
		)

		if err != nil {
			errMsg := ":red_circle: La réservation a échoué"
			if bView, ok := newView.(*views.BrowseView); ok {
				bView.Error = &errMsg
			}
		} else {
			bView := newView.(*views.BrowseView)
			bView.Phase = 2

			err = s.store.SetSlackState(result.User.ID, views.ViewType(bView), bView)

			if err != nil {
				return err
			}

			blocks := views.RenderView(bView)
			return s.SendToSlack(result.ResponseURL, blocks)
		}
	case *views.ReservationCmd:
		reservations, err := s.fetchReservations(*user)
		rView := newView.(*views.ReservationView)

		if err != nil {
			errMsg := ":red_circle: Impossible de charger les réservations"
			rView.Error = &errMsg
		} else {
			rView.Reservations = &reservations
		}

	case *views.CancelReservationCmd:
		rView := newView.(*views.ReservationView)
		err := s.cancelReservation(*user, *c.ReservationId)
		if err != nil {
			errMsg := ":red_circle: Impossible d'annuler la réservation les réservations"
			rView.Error = &errMsg
		} else {
			rView.Phase = 1
		}
	}

	err = s.store.SetSlackState(result.User.ID, views.ViewType(newView), newView)
	if err != nil {
		return err
	}

	blocks := views.RenderView(newView)
	return s.SendToSlack(result.ResponseURL, blocks)
}

func (s *SlackService) SetSlackState(slackUserId, messageType string, state any) error {
	return s.store.SetSlackState(slackUserId, messageType, state)
}

func (s *SlackService) SendToSlack(responseUrl string, blocks slack.Block) error {
	jsonBlocks, err := json.Marshal(blocks)

	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", responseUrl, bytes.NewBuffer(jsonBlocks))

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	_, err = http.DefaultClient.Do(req)

	return err
}

func (s *SlackService) getRoomAvailabilities(
	user storage.User,
	nbPeople, duration int,
	dateTime time.Time,
) ([]models.Room, error) {

	apiClient := api.NewApi()

	payload := api.CosoftAvailabilityPayload{
		DateTime: dateTime,
		NbPeople: nbPeople,
		Duration: duration,
	}

	rooms, err := apiClient.GetAvailableRooms(user.WAuth, user.WAuthRefresh, payload)

	if err != nil {
		return nil, err
	}

	if len(rooms) == 0 {
		return nil, fmt.Errorf(":red_circle: Aucune salle disponible")
	}

	return rooms, nil
}

func (s *SlackService) bookRoom(
	user storage.User,
	nbPeople, duration int,
	pickedRoom models.Room,
	dateTime time.Time,
) error {

	payload := api.CosoftBookingPayload{
		CosoftAvailabilityPayload: api.CosoftAvailabilityPayload{
			NbPeople: nbPeople,
			Duration: duration,
			DateTime: dateTime,
		},
		UserCredits: user.Credits,
		Room:        pickedRoom,
	}

	apiClient := api.NewApi()

	err := apiClient.BookRoom(user.WAuth, user.WAuthRefresh, payload)

	if err != nil {
		return err
	}

	return nil
}

func (s *SlackService) fetchReservations(user storage.User) ([]api.Reservation, error) {
	apiClient := api.NewApi()
	bookings, err := apiClient.GetFutureBookings(user.WAuth, user.WAuthRefresh)

	if err != nil {
		return nil, err
	}

	return bookings.Data, nil
}

func (s *SlackService) cancelReservation(
	user storage.User,
	reservationId string,
) error {
	apiClient := api.NewApi()

	return apiClient.CancelBooking(user.WAuth, user.WAuthRefresh, reservationId)
}
