package agents

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	OpenAIAPIKey     string `json:"OPENAI_API_KEY"`
	OpenrouterAPIKey string `json:"OPENROUTER_API_KEY"`
	SerpAPIKey       string `json:"SERP_API_KEY"`
}

func NewConfig() *Config {
	bytes, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	json.Unmarshal(bytes, &config)

	return &config
}
