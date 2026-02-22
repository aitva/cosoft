package models

import (
	"cosoft-cli/internal/api"
	"cosoft-cli/internal/storage"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ReservationState struct {
	Phase             int
	Reservations      *[]api.Reservation
	PickedReservation *api.Reservation
	ReservationId     *string
	BookingStarted    bool
	Error             *string
}

func newReservationState(store *storage.Store, userID string) (State, error) {
	user, err := store.GetUserData(&userID)
	if err != nil {
		return &LoginState{
			Error: ptr(errInternalError),
		}, fmt.Errorf("get user data: %v", err)
	}

	apiClient := api.NewApi()
	bookings, err := apiClient.GetFutureBookings(user.WAuth, user.WAuthRefresh)
	if err != nil {
		return &ReservationState{
			Error: ptr(":red_circle: Impossible de charger les réservations"),
		}, fmt.Errorf("get future bookings: %v", err)
	}

	return &ReservationState{
		Reservations: &bookings.Data,
	}, nil
}

func (s *ReservationState) Type() string { return reservationStateType }

func (s *ReservationState) Update(store *storage.Store, params UpdateParams) (State, error) {
	if params.ActionID == "back" {
		return NewLandingState(store, params.UserID)
	}

	if params.ActionID == "cancel" {
		return s.cancel(store, cancelReservationParams{
			UserID:        params.UserID,
			ReservationID: *s.ReservationId,
		})
	}

	// Action id is the uuid of a selected booking.
	if err := uuid.Validate(params.ActionID); err == nil {
		s.ReservationId = &params.ActionID

		// Resetting "BookingStarted" to avoid being blocked.
		s.BookingStarted = false

		for _, reservation := range *s.Reservations {
			if reservation.OrderResourceRentId == params.ActionID {
				s.PickedReservation = &reservation
				break
			}
		}

		if s.PickedReservation == nil {
			fmt.Println("Could not find a picked reservation")
			return s, nil
		}

		// Ensure the reservation hasn't already started
		bookinStartsAt, err := time.ParseInLocation("2006-01-02T15:04:05", s.PickedReservation.Start, time.Local)

		if err != nil {
			fmt.Println(err)
			return s, nil
		}

		if bookinStartsAt.Before(time.Now()) {
			s.BookingStarted = true
		}
	}

	return s, nil
}

func (s *ReservationState) Next() bool { return false }

type cancelReservationParams struct {
	UserID        string
	ReservationID string
}

func (s *ReservationState) cancel(store *storage.Store, params cancelReservationParams) (State, error) {
	var errCannotCancel = ":red_circle: Impossible d'annuler la réservation"

	user, err := store.GetUserData(&params.UserID)
	if err != nil {
		// TODO: redirect the user to the login page and display an error?
		s.Error = &errCannotCancel
		return s, fmt.Errorf("get user data: %v", err)
	}

	apiClient := api.NewApi()
	err = apiClient.CancelBooking(user.WAuth, user.WAuthRefresh, params.ReservationID)
	if err != nil {
		s.Error = &errCannotCancel
		return s, fmt.Errorf("cancel booking: %v", err)
	}

	// TODO: Should it be replaced by ReservationCancelledState?
	s.Phase = 1
	return s, nil
}
