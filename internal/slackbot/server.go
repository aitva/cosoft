package slackbot

import (
	"cosoft-cli/internal/slackbot/services"
	"cosoft-cli/internal/slackbot/views"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

func (b *Bot) StartServer() {
	s := http.Server{
		Addr: ":11111",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Request received", slog.String("method", r.Method), slog.String("url", r.URL.String()))

			switch r.URL.String() {
			case "/book":
				b.handleRequests(w, r)
			case "/interact":
				b.handleInteractions(w, r)
			default:
				fmt.Println("Unknown URL", r.URL.String())
			}
		}),
	}

	slog.Info("Server is starting...")

	err := s.ListenAndServe()

	if err != nil {
		slog.Error("failed to listen and serve", "err", err.Error())
	}
}

func (b *Bot) handleRequests(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		return
	}

	slackRequest := &services.Request{
		Command:     r.Form.Get("command"),
		Text:        r.Form.Get("text"),
		UserId:      r.Form.Get("user_id"),
		ResponseUrl: r.Form.Get("response_url"),
		TriggerId:   r.Form.Get("trigger_id"),
	}

	// Clear out user's old slack states
	err = b.service.ClearUserStates(slackRequest)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Build and send view in the background.
	go func() {
		state, err := b.service.AuthGuard(slackRequest)
		if err != nil {
			fmt.Println(err)
		}

		blocks := views.RenderState(state)
		err = b.service.SendToSlack(slackRequest.ResponseUrl, blocks)
		if err != nil {
			fmt.Println(err)
			return
		}
	}()

	// Return a temporary view.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"response_type":"ephemeral","text":"Chargement en cours..."}`))
}

func (b *Bot) handleInteractions(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
		return
	}

	payload := r.Form.Get("payload")
	go func() {
		err := b.service.HandleInteraction(payload)

		if err != nil {
			fmt.Println(err)
		}
	}()

	w.WriteHeader(http.StatusOK)
}

func debug(payload any) {
	f, _ := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	fmt.Fprintf(f, "%v\n\n", payload)
}
