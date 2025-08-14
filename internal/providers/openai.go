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

type OpenAI struct{ APIKey, BaseURL, ModelID string }

func (o *OpenAI) Generate(ctx context.Context, prompt string, params map[string]any) (string, Usage, error) {
	if o.APIKey == "" {
		o.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if o.BaseURL == "" {
		o.BaseURL = "https://api.openai.com"
	}
	model := strings.TrimPrefix(o.ModelID, "openai-")
	url := o.BaseURL + "/v1/chat/completions"

	body, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
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
		return "", Usage{}, fmt.Errorf("openai status %d", resp.StatusCode)
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", Usage{}, err
	}
	text := ""
	if len(out.Choices) > 0 {
		text = out.Choices[0].Message.Content
	}
	return text, Usage{
		PromptTokens:     out.Usage.PromptTokens,
		CompletionTokens: out.Usage.CompletionTokens,
		TotalTokens:      out.Usage.TotalTokens,
	}, nil
}
