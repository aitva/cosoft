package models

import (
	"cmp"
	"cosoft-cli/internal/api"
	"cosoft-cli/internal/common"
	"cosoft-cli/internal/storage"
	"cosoft-cli/shared/models"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type BrowseState struct {
	Phase      int
	NbPeople   string
	Duration   string
	Date       string
	Time       string
	Rooms      *[]models.Room
	PickedRoom *models.Room
	Error      *string
}

type browsePayload struct {
	Time struct {
		Time struct {
			Type         string `json:"type"`
			SelectedTime string `json:"selected_time"`
		} `json:"time"`
	} `json:"time"`
	Date struct {
		Date struct {
			Type         string `json:"type"`
			SelectedDate string `json:"selected_date"`
		} `json:"date"`
	} `json:"date"`
	Duration struct {
		Duration struct {
			Type           string `json:"type"`
			SelectedOption struct {
				Value string `json:"value"`
			} `json:"selected_option"`
		} `json:"duration"`
	} `json:"duration"`
	NbPeople struct {
		NbPeople struct {
			Type           string `json:"type"`
			SelectedOption struct {
				Value string `json:"value"`
			} `json:"selected_option"`
		} `json:"nbPeople"`
	} `json:"nbPeople"`
}

type pickedRoomPayload struct {
	PickRoom struct {
		PickRoom struct {
			Type           string `json:"type"`
			SelectedOption struct {
				Value string `json:"value"`
			} `json:"selected_option"`
		} `json:"pick-room"`
	} `json:"pick-room"`
}

func (s *BrowseState) Type() string { return browseStateType }

func (s *BrowseState) Update(store *storage.Store, params UpdateParams) (State, error) {
	switch params.ActionID {
	case "cancel":
		return NewLandingState(store, params.UserID)

	case "browse":
		return s.browseRooms(store, params)

	case "pick-room":
		return s.pickRoom(store, params)

	case "book":
		return s.bookRoom(store, params)

	case "back":
		s.Phase = 0
		return s, nil

	default:
		// TODO: remove the following code?
		//if strings.HasPrefix(params.ActionID, "book-") {
		//	// A room has been picked
		//	// Return this for now.
		//	return s, nil
		//}
		return s, fmt.Errorf("unexpected action ID: %v", params.ActionID)
	}
}

func (s *BrowseState) Next() bool { return false }

type browseStateQuery struct {
	Time     time.Time
	NbPeople int
	Duration int
}

func (s *BrowseState) parseQuery() (browseStateQuery, error) {
	// Parse and validate time.
	t, err := time.Parse("2006-01-02 15:04", s.Date+" "+s.Time)
	if err != nil {
		s.Error = ptr(":warning: Veuillez choisir une date et une heure valide.")
		return browseStateQuery{}, fmt.Errorf("parse date and time: %v", err)
	}
	if t.Before(time.Now()) {
		s.Error = ptr(":warning: Veuillez choisir une date dans le futur")
		return browseStateQuery{}, nil
	}
	if t.Minute()%15 != 0 {
		s.Error = ptr(":warning: Veuillez choisir un quart d'heure (14h00, 15h15, 16h30, 17h45....)")
		return browseStateQuery{}, nil
	}

	nbPeople, err := strconv.Atoi(s.NbPeople)
	if err != nil {
		s.Error = ptr(":warning: Veuillez choisir un nombre de personnes valide.")
		return browseStateQuery{}, fmt.Errorf("parse number of people: %v", err)
	}

	duration, err := strconv.Atoi(s.Duration)
	if err != nil {
		s.Error = ptr(":warning: Veuillez choisir une durée valide.")
		return browseStateQuery{}, fmt.Errorf("parse duration: %v", err)
	}

	return browseStateQuery{
		Time:     t,
		NbPeople: nbPeople,
		Duration: duration,
	}, nil
}

func (s *BrowseState) browseRooms(store *storage.Store, params UpdateParams) (State, error) {
	// Parse the payload.
	var values browsePayload
	err := json.Unmarshal(params.Values, &values)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Load duration and number of people.
	s.Duration = values.Duration.Duration.SelectedOption.Value
	s.NbPeople = values.NbPeople.NbPeople.SelectedOption.Value
	if s.NbPeople == "" || s.Duration == "" {
		s.Error = ptr(":warning: Tous les champs sont requis")
		return s, nil
	}

	// Load date and time or fallback to their default values.
	//
	// Slack only sends values changed by the user.
	s.Date = cmp.Or(values.Date.Date.SelectedDate, time.Now().Format(time.DateOnly))
	s.Time = cmp.Or(values.Time.Time.SelectedTime, common.GetClosestQuarterHour().Format("15:04"))

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

func (s *BrowseState) pickRoom(store *storage.Store, params UpdateParams) (State, error) {
	var pickedRoom pickedRoomPayload
	err := json.Unmarshal(params.Values, &pickedRoom)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	roomId := pickedRoom.PickRoom.PickRoom.SelectedOption.Value
	for _, r := range *s.Rooms {
		if r.Id == roomId {
			s.PickedRoom = &r
			break
		}
	}
	if s.PickedRoom == nil {
		return s, fmt.Errorf("fail to find room: %v", roomId)
	}

	return s, nil
}

func (s *BrowseState) bookRoom(store *storage.Store, params UpdateParams) (State, error) {
	query, err := s.parseQuery()
	if err != nil {
		return s, fmt.Errorf("parse query: %v", err)
	}

	user, err := store.GetUserData(&params.UserID)
	if err != nil {
		// TODO: redirect the user to the login page and display an error?
		return s, fmt.Errorf("get user data: %v", err)
	}

	apiClient := api.NewApi()
	err = apiClient.BookRoom(user.WAuth, user.WAuthRefresh, api.CosoftBookingPayload{
		CosoftAvailabilityPayload: api.CosoftAvailabilityPayload{
			NbPeople: query.NbPeople,
			Duration: query.Duration,
			DateTime: query.Time,
		},
		UserCredits: user.Credits,
		Room:        *s.PickedRoom,
	})
	if err != nil {
		s.Error = ptr(":red_circle: La réservation a échoué")
		return s, fmt.Errorf("book room: %v", err)
	}

	s.Phase = 2
	return s, nil
}
