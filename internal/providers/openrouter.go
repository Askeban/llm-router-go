package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type OpenRouter struct{ APIKey, ModelID string }

func (o *OpenRouter) Generate(ctx context.Context, prompt string, params map[string]any) (string, Usage, error) {
	if o.APIKey == "" {
		o.APIKey = os.Getenv("OPENROUTER_API_KEY")
	}
	model := strings.TrimPrefix(o.ModelID, "openrouter-")
	url := "https://openrouter.ai/api/v1/chat/completions"

	body, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	cli := &http.Client{Timeout: 60 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return "", Usage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", Usage{}, fmt.Errorf("openrouter status %d", resp.StatusCode)
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", Usage{}, err
	}
	text := ""
	if len(out.Choices) > 0 {
		text = out.Choices[0].Message.Content
	}
	return text, Usage{}, nil
}
