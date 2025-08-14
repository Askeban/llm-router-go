package classifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	Base string
	http *http.Client
}

func New(base string) *Client {
	return &Client{Base: base, http: &http.Client{Timeout: 8 * time.Second}}
}

type req struct {
	Prompt string `json:"prompt"`
}
type resp struct {
	Category, Difficulty string
	Confidence           float64
}

func (c *Client) Classify(ctx context.Context, prompt string) (string, string, error) {
	b, _ := json.Marshal(req{Prompt: prompt})
	rq, _ := http.NewRequestWithContext(ctx, "POST", c.Base+"/classify", bytes.NewReader(b))
	rq.Header.Set("Content-Type", "application/json")
	rs, err := c.http.Do(rq)
	if err != nil {
		return "", "", err
	}
	defer rs.Body.Close()
	if rs.StatusCode != 200 {
		return "", "", fmt.Errorf("classifier status %d", rs.StatusCode)
	}
	var out resp
	if err := json.NewDecoder(rs.Body).Decode(&out); err != nil {
		return "", "", err
	}
	return out.Category, out.Difficulty, nil
}
