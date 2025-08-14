package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct{ Port, ClassifierURL, DatabasePath, ModelProfilesPath, OpenAIKey, AnthropicKey, GoogleAPIKey, OpenRouterKey, OpenAIBaseURL string }

func Load() (*Config, error) {
	c := &Config{Port: getenv("PORT_DEFAULT", "8080"), ClassifierURL: getenv("CLASSIFIER_URL", "http://classifier:5000"), DatabasePath: getenv("SQLITE_PATH", "data/router.db"), ModelProfilesPath: getenv("MODEL_PROFILES_PATH", "configs/models.json"), OpenAIKey: os.Getenv("OPENAI_API_KEY"), AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"), GoogleAPIKey: os.Getenv("GOOGLE_API_KEY"), OpenRouterKey: os.Getenv("OPENROUTER_API_KEY"), OpenAIBaseURL: getenv("OPENAI_BASE_URL", "https://api.openai.com")}
	if p := os.Getenv("APP_CONFIG_JSON"); p != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", p, err)
		}
		if err := json.Unmarshal(b, c); err != nil {
			return nil, fmt.Errorf("json %s: %w", p, err)
		}
	}
	return c, nil
}
func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
