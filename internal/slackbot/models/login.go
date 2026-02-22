package models

import (
	"cosoft-cli/internal/api"
	"cosoft-cli/internal/storage"
	"encoding/json"
	"fmt"
)

type LoginState struct {
	Email    string
	Password string
	Error    *string
}

type loginValues struct {
	Email struct {
		Email struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"email"`
	} `json:"email"`
	Password struct {
		Password struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"password"`
	} `json:"password"`
}

func (s *LoginState) Type() string { return loginStateType }

func (s *LoginState) Update(store *storage.Store, params UpdateParams) (State, error) {
	var values loginValues

	err := json.Unmarshal(params.Values, &values)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	s.Email = values.Email.Email.Value
	s.Password = values.Password.Password.Value
	s.Error = nil

	if s.Email == "" || s.Password == "" {
		tmp := ":warning: Tous les champs sont requis"
		s.Error = &tmp
		return s, nil
	}

	err = login(store, loginParams{
		Email:    s.Email,
		Password: s.Password,
		UserID:   params.UserID,
	})
	if err != nil {
		tmp := ":red_circle: Identifiant / mot de passe incorrect"
		s.Error = &tmp
		return s, nil
	}

	return s, nil
}

func (s *LoginState) Next() bool { return false }

type loginParams struct {
	Email    string
	Password string
	UserID   string
}

func login(store *storage.Store, params loginParams) error {
	apiClient := api.NewApi()

	response, err := apiClient.Login(&api.LoginPayload{
		Email:    params.Email,
		Password: params.Password,
	})
	if err != nil {
		return err
	}

	return store.SetUser(
		response,
		response.JwtToken,
		response.RefreshToken,
		&params.UserID,
	)
}
