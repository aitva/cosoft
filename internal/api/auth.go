package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type UserResponse struct {
	JwtToken     string  `json:"JwtToken"`
	RefreshToken string  `json:"-"`
	Id           string  `json:"Id"`
	FirstName    string  `json:"FirstName"`
	LastName     string  `json:"LastName"`
	Email        string  `json:"Email"`
	Credits      float64 `json:"Credits"`
}

type AuthPayload struct {
	IsAuth  bool          `json:"isAuth"`
	Message string        `json:"Message"`
	User    *UserResponse `json:"User"`
}

func (a *Api) Login(payload *LoginPayload) (*UserResponse, error) {
	values := map[string]string{
		"email":    payload.Email,
		"password": payload.Password,
	}

	jsonValues, err := json.Marshal(values)

	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(
		apiUrl+"/users/login",
		"application/json",
		bytes.NewBuffer(jsonValues),
	)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	response := AuthPayload{}

	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if !response.IsAuth || response.User == nil {
		return nil, fmt.Errorf("Wrong username / password")
	}

	// Extract refresh token from Set-Cookie header
	cookies := resp.Header.Values("Set-Cookie")
	refreshToken := extractRefreshToken(cookies)
	response.User.RefreshToken = refreshToken

	return response.User, nil
}

// extractRefreshToken extracts w_auth_refresh from Set-Cookie headers
func extractRefreshToken(cookies []string) string {
	for _, cookie := range cookies {
		if strings.Contains(cookie, "w_auth_refresh=") {
			parts := strings.Split(cookie, ";")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "w_auth_refresh=") {
					return strings.TrimPrefix(part, "w_auth_refresh=")
				}
			}
		}
	}
	return ""
}

func (a *Api) GetAuth(wAuth, wAuthRefresh string) error {
	req, client, err := a.prepareHeaderCookies(
		wAuth,
		wAuthRefresh,
		"GET",
		fmt.Sprintf("%s/users/auth", apiUrl),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	response := AuthPayload{}
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if !response.IsAuth || response.User == nil {
		return fmt.Errorf("Wrong username / password")
	}

	return nil
}

func (a *Api) GetCredits(wAuth, wAuthRefresh string) (float64, error) {

	req, client, err := a.prepareHeaderCookies(
		wAuth,
		wAuthRefresh,
		"GET",
		fmt.Sprintf("%s/users/auth", apiUrl),
		nil,
	)

	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)

	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	response := AuthPayload{}

	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	return response.User.Credits, nil
}

func (a *Api) Logout(wAuth, wAuthRefresh string) error {
	req, client, err := a.prepareHeaderCookies(
		wAuth,
		wAuthRefresh,
		"POST",
		fmt.Sprintf("%s/users/logout", apiUrl),
		nil,
	)

	if err != nil {
		return err
	}

	_, err = client.Do(req)

	if err != nil {
		return err
	}

	return err
}
