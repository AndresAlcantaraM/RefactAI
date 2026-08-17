package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GeminiApiKey string
	GeminiModel  string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY is not configured")
	}

	return &Config{
		GeminiApiKey: apiKey,
		GeminiModel:  "gemini-2.5-flash",
	}, nil
}
