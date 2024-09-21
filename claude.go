package anthropic

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/citizenadam/go-synochat"
	"github.com/citizenadam/anthropic"
)

type WebhookPayload struct {
	Token       string `json:"token"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	PostID      string `json:"post_id"`
	Timestamp   string `json:"timestamp"`
	Text        string `json:"text"`
	TriggerWord string `json:"trigger_word"`
}

func main() {
	// Initialize the Synology Chat client
	baseURL := os.Getenv("SYNOLOGY_CHAT_BASE_URL")
	if baseURL == "" {
		log.Fatal("SYNOLOGY_CHAT_BASE_URL environment variable is not set")
	}

	outgoingToken := os.Getenv("SYNOLOGY_CHAT_OUTGOING_TOKEN")
	if outgoingToken == "" {
		log.Fatal("SYNOLOGY_CHAT_OUTGOING_TOKEN environment variable is not set")
	}

	synoClient, err := synochat.NewClient(baseURL)
	if err != nil {
		log.Fatalf("Failed to create Synology Chat client: %v", err)
	}

	// Initialize the Anthropic client
	anthropicAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicAPIKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is not set")
	}

	anthropicModel := os.Getenv("ANTHROPIC_MODEL")
	if anthropicModel == "" {
		anthropicModel = "claude-3-5-sonnet-20240620" // Default model
	}

	anthropicClient := anthropic.NewClient(anthropicAPIKey, anthropicModel)

	// Set up the HTTP handler for incoming webhooks
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var payload WebhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "Failed to parse JSON payload", http.StatusBadRequest)
			return
		}

		// Verify the incoming token (optional, but recommended for security)
		if payload.Token != outgoingToken {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Forward the message to Anthropic
		anthropicResponse, err := anthropicClient.SendMessage(payload.Text)
		if err != nil {
			log.Printf("Failed to get response from Anthropic: %v", err)
			http.Error(w, "Failed to process message", http.StatusInternalServerError)
			return
		}

		// Send the Anthropic response back to Synology Chat
		err = synoClient.SendMessage(&synochat.ChatMessage{
			Text: anthropicResponse,
		}, outgoingToken)

		if err != nil {
			log.Printf("Failed to send message to Synology Chat: %v", err)
			http.Error(w, "Failed to send response", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	// Start the HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
