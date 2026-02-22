package models

import (
	"cosoft-cli/internal/api"
	"cosoft-cli/internal/common"
	"cosoft-cli/internal/storage"
	"cosoft-cli/shared/models"
	shared "cosoft-cli/shared/models"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type QuickBookState struct {
	Phase      int
	NbPeople   string
	Duration   string
	Rooms      *[]shared.Room
	PickedRoom *shared.Room
	Error      *string
}

type QuickBookValues struct {
	Duration struct {
		Duration struct {
			SelectedOption struct {
				Value string `json:"value"`
			} `json:"selected_option"`
		} `json:"duration"`
	} `json:"duration"`
	NbPeople struct {
		NbPeople struct {
			SelectedOption struct {
				Value string `json:"value"`
			} `json:"selected_option"`
		} `json:"nbPeople"`
	} `json:"nbPeople"`
}

func (s *QuickBookState) Type() string { return quickBookStateType }

func (s *QuickBookState) Update(store *storage.Store, params UpdateParams) (State, error) {
	switch params.ActionID {
	case "cancel":
		return NewLandingState(store, params.UserID)

	case "quick-book":
		if s.Phase == 2 {
			return s.book(store, params)
		}
		return s.pickRoom(store, params)

	default:
		return s, nil
	}
}

func (s *QuickBookState) Next() bool {
	// Another update is available if we are in phase 1 and there is no error.
	//
	// TODO: there must be an easier way.
	return s.Phase == 1 && s.Error == nil
}

func (s *QuickBookState) pickRoom(store *storage.Store, params UpdateParams) (State, error) {
	var values QuickBookValues
	err := json.Unmarshal(params.Values, &values)
	if err != nil {
		return s, fmt.Errorf("unmarshall values: %v", err)
	}

	// TODO: remove this assignment?
	s.Error = nil

	// Load duration and the number of people.
	s.Duration = values.Duration.Duration.SelectedOption.Value
	s.NbPeople = values.NbPeople.NbPeople.SelectedOption.Value
	if s.NbPeople == "" || s.Duration == "" {
		s.Error = ptr(":warning: Tous les champs sont requis")
		return s, nil
	}

	query, err := s.parseQuery()
	if err != nil {
		// s.Error is set by parseQuery.
		return s, fmt.Errorf("parse query: %v", err)
	}

	user, err := store.GetUserData(&params.UserID)
	if err != nil {
		// TODO: redirect the user to the login page and display an error?
		return s, fmt.Errorf("get user data: %v", err)
	}

	apiClient := api.NewApi()
	rooms, err := apiClient.GetAvailableRooms(user.WAuth, user.WAuthRefresh, api.CosoftAvailabilityPayload{
		DateTime: query.Time,
		NbPeople: query.NbPeople,
		Duration: query.Duration,
	})
	if err != nil {
		s.Error = ptr(":red_circle: La réservation a échoué")
		return s, fmt.Errorf("get available rooms: %v", err)
	}
	if len(rooms) == 0 {
		s.Error = ptr(":red_circle: Aucune salle disponible")
		return s, nil
	}

	s.Phase = 1
	s.Rooms = &rooms
	return s, nil
}

func (s *QuickBookState) book(store *storage.Store, params UpdateParams) (State, error) {
	var values QuickBookValues
	err := json.Unmarshal(params.Values, &values)
	if err != nil {
		return s, fmt.Errorf("unmarshall values: %v", err)
	}

	// TODO: remove this assignment?
	s.Error = nil

	// Load duration and the number of people.
	s.Duration = values.Duration.Duration.SelectedOption.Value
	s.NbPeople = values.NbPeople.NbPeople.SelectedOption.Value
	if s.NbPeople == "" || s.Duration == "" {
		s.Error = ptr(":warning: Tous les champs sont requis")
		return s, nil
	}

	query, err := s.parseQuery()
	if err != nil {
		// s.Error is set by parseQuery.
		return s, fmt.Errorf("parse query: %v", err)
	}

	user, err := store.GetUserData(&params.UserID)
	if err != nil {
		// TODO: redirect the user to the login page and display an error?
		return s, fmt.Errorf("get user data: %v", err)
	}

	var pickedRoom *models.Room
	for _, room := range *s.Rooms {
		if room.NbUsers >= query.NbPeople {
			pickedRoom = &room
			break
		}
	}
	if pickedRoom == nil {
		s.Error = ptr(":red_circle: Aucune salle disponible")
		return s, nil
	}
	if user.Credits < pickedRoom.Price {
		s.Error = ptr(":red_circle: Pas assez de crédits pour faire une réservation")
		return s, nil
	}

	apiClient := api.NewApi()
	err = apiClient.BookRoom(user.WAuth, user.WAuthRefresh, api.CosoftBookingPayload{
		CosoftAvailabilityPayload: api.CosoftAvailabilityPayload{
			NbPeople: query.NbPeople,
			Duration: query.Duration,
			DateTime: query.Time,
		},
		UserCredits: user.Credits,
		Room:        *pickedRoom,
	})
	if err != nil {
		s.Error = ptr(":red_circle: La réservation a échoué")
		return s, fmt.Errorf("book room: %v", err)
	}

	s.PickedRoom = pickedRoom
	s.Phase = 2
	return s, nil
}

type quickBookQuery struct {
	Time     time.Time
	NbPeople int
	Duration int
}

func (s *QuickBookState) parseQuery() (quickBookQuery, error) {
	nbPeople, err := strconv.Atoi(s.NbPeople)
	if err != nil {
		s.Error = ptr(":warning: Veuillez choisir un nombre de personnes valide.")
		return quickBookQuery{}, fmt.Errorf("parse number of people: %v", err)
	}

	duration, err := strconv.Atoi(s.Duration)
	if err != nil {
		s.Error = ptr(":warning: Veuillez choisir une durée valide.")
		return quickBookQuery{}, fmt.Errorf("parse duration: %v", err)
	}

	return quickBookQuery{
		Time:     common.GetClosestQuarterHour(),
		NbPeople: nbPeople,
		Duration: duration,
	}, nil
}
