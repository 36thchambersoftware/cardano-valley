package discord

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slog"
)

func Webhook() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// TODO replace with webhook params struct
	data := discordgo.WebhookParams{Content: "cow tipping!", Username: "notabot"}
	params, err := json.Marshal(data)
	if err != nil {
		slog.Error("cardano-valley", "PACKAGE", "DISCORD", "ERROR", err)
	}
	// data := strings.NewReader("{username: 'Bob',content: 'Hi Dylan',}")

	// Create a new request
	request, err := http.NewRequest(http.MethodPost, DISCORD_WEBHOOK_URL, bytes.NewBuffer(params))
	if err != nil {
		slog.Error("cardano-valley", "PACKAGE", "DISCORD", "request", err)
		return
	}

	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		slog.Error("cardano-valley", "PACKAGE", "DISCORD", "response", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNoContent {
		slog.Warn("cardano-valley", "PACKAGE", "DISCORD", "INFO", "You are not waiting for a response. Add ?wait=true to webhook url")
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Error("cardano-valley", "PACKAGE", "DISCORD", "body", body, "error", err)
		return
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		slog.Error("cardano-valley", "PACKAGE", "DISCORD", "STATUS", response.StatusCode, "ERROR", string(body))
		return
	}
}
