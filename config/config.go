package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// PocketBase External Server
	PocketBaseURL   string // PocketBase server URL (e.g., http://192.168.100.100:8090)
	PocketBaseToken string // Auth token for API access

	// Telegram Bot
	TelegramBotToken string
	AuthorizedChatID string
}

func LoadConfig() (*Config, error) {
	cwd, _ := os.Getwd()
	log.Printf("Current working directory: %s", cwd)

	if _, err := os.Stat(".env"); err != nil {
		log.Printf("os.Stat(.env) error: %v", err)
	} else {
		log.Println(".env file exists according to os.Stat")
	}

	err := godotenv.Load()
	if err != nil {
		log.Printf("godotenv.Load() error: %v", err)
	}

	// Get PocketBase URL (required)
	pbURL := os.Getenv("POCKETBASE_URL")
	if pbURL == "" {
		pbURL = "http://192.168.100.100:8090" // Default external server
	}

	return &Config{
		PocketBaseURL:    pbURL,
		PocketBaseToken:  os.Getenv("POCKETBASE_TOKEN"),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		AuthorizedChatID: os.Getenv("AUTHORIZED_CHAT_ID"),
	}, nil
}
