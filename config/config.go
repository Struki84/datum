package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	OpenAIAPIKey     string `json:"OPENAI_API_KEY"`
	OpenrouterAPIKey string `json:"OPENROUTER_API_KEY"`
	SerpAPIKey       string `json:"SERP_API_KEY"`
	PineconeHost     string `json:"PineconeHost"`
	PineconeAPIKey   string `json:"PineconeAPIKey"`
}

func New() *Config {
	bytes, err := os.ReadFile("./config/my.config.json")
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	json.Unmarshal(bytes, &config)

	return &config
}
