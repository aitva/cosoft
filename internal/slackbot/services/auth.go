package services

import (
	"cosoft-cli/internal/api"
	"cosoft-cli/internal/slackbot/models"
	"fmt"
)

type Request struct {
	UserId      string
	Command     string
	Text        string
	ResponseUrl string
	TriggerId   string
}

func (s *SlackService) AuthGuard(request *Request) (models.State, error) {
	cookies, err := s.store.HasActiveToken(&request.UserId)
	if err != nil {
		return &models.LoginState{}, fmt.Errorf("has active token: %v", err)
	}

	// Fast path if there are no cookies.
	if cookies == nil {
		return &models.LoginState{}, nil
	}

	// TODO: why are we doing this?
	apiClient := api.NewApi()
	err = apiClient.GetAuth(cookies.WAuth, cookies.WAuthRefresh)
	if err != nil {
		return &models.LoginState{}, fmt.Errorf("get auth: %v", err)
	}

	state, err := models.NewLandingState(s.store, request.UserId)
	if err != nil {
		return state, fmt.Errorf("new landing state: %v", err)
	}

	return state, nil
}

func (s *SlackService) ClearUserStates(request *Request) error {
	return s.store.ResetUserSlackState(request.UserId)
}
