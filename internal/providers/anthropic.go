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

type Anthropic struct{ APIKey, ModelID string }

func (a *Anthropic) Generate(ctx context.Context, prompt string, params map[string]any) (string, Usage, error) {
	if a.APIKey == "" {
		a.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	model := strings.TrimPrefix(a.ModelID, "anthropic-")
	url := "https://api.anthropic.com/v1/messages"

	body, _ := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	cli := &http.Client{Timeout: 60 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return "", Usage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", Usage{}, fmt.Errorf("anthropic status %d", resp.StatusCode)
	}

	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", Usage{}, err
	}
	text := ""
	if len(out.Content) > 0 {
		text = out.Content[0].Text
	}
	return text, Usage{
		PromptTokens:     out.Usage.InputTokens,
		CompletionTokens: out.Usage.OutputTokens,
		TotalTokens:      out.Usage.InputTokens + out.Usage.OutputTokens,
	}, nil
}
