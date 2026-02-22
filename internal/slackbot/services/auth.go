package services

import (
	"cosoft-cli/internal/api"
	"cosoft-cli/internal/slackbot/views"
	"cosoft-cli/internal/storage"
	"cosoft-cli/shared/models"
)

func (s *SlackService) AuthGuard(request models.Request) (*views.LoginView, error) {
	cookies, err := s.store.HasActiveToken(&request.UserId)

	if err != nil || cookies == nil {

		loginView := &views.LoginView{
			Email:    "",
			Password: "",
		}

		err := s.store.SetSlackState(request.UserId, "login", loginView)

		if err != nil {
			return nil, err
		}

		return loginView, nil
	}

	apiClient := api.NewApi()
	err = apiClient.GetAuth(cookies.WAuth, cookies.WAuthRefresh)

	return nil, nil
}

func (s *SlackService) ClearUserStates(request models.Request) error {
	return s.store.ResetUserSlackState(request.UserId)
}

func (s *SlackService) GetUserData(userId string) (*storage.User, error) {
	return s.store.GetUserData(&userId)
}

func (s *SlackService) RefreshAndGetUser(slackUserId string) (*storage.User, error) {
	_, err := s.store.UpdateCredits(&slackUserId)

	if err != nil {
		return nil, err
	}

	user, err := s.GetUserData(slackUserId)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *SlackService) LogInUser(email, password, slackUserId string) error {
	apiClient := api.NewApi()

	response, err := apiClient.Login(&api.LoginPayload{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return err
	}

	return s.store.SetUser(
		response,
		response.JwtToken,
		response.RefreshToken,
		&slackUserId,
	)
}
