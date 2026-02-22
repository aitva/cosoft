package services

import (
	"bytes"
	"cosoft-cli/internal/slackbot/models"
	"cosoft-cli/internal/slackbot/views"
	"cosoft-cli/internal/ui/slack"
	"encoding/json"
	"fmt"
	"net/http"
)

type interactionDiscovery struct {
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	State struct {
		Values json.RawMessage `json:"values"`
	} `json:"state"`
	ResponseURL string `json:"response_url"`
	Actions     []struct {
		ActionID string `json:"action_id"`
	}
}

func (s *SlackService) HandleInteraction(payload string) error {
	var tmp interactionDiscovery
	err := json.Unmarshal([]byte(payload), &tmp)
	if err != nil {
		return err
	}

	state, err := models.LoadState(s.store, tmp.User.ID)
	if err != nil {
		// TODO: log error?
		return fmt.Errorf("no active view for user %s", tmp.User.ID)
	}

	for state.Next() {
		state, err = state.Update(s.store, models.UpdateParams{
			ActionID: tmp.Actions[0].ActionID,
			Values:   tmp.State.Values,
		})
		if err != nil {
			// TODO: log error?
		}

		err = models.SaveState(s.store, tmp.User.ID, state)
		if err != nil {
			// TODO: log error?
		}

		blocks := views.RenderState(state)
		err = s.SendToSlack(tmp.ResponseURL, blocks)
		if err != nil {
			// TODO: log error?
		}
	}

	return nil
}

func (s *SlackService) SendToSlack(responseUrl string, blocks slack.Block) error {
	body, err := json.Marshal(blocks)
	if err != nil {
		return fmt.Errorf("mashal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", responseUrl, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("new request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do http request: %v", err)
	}

	return nil
}
