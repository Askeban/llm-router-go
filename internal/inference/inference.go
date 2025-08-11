package inference

import (
	"context"
	"errors"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// SendPrompt sends the given prompt to the specified model and returns the model's
// response. Currently only OpenAI GPT models are supported. The OpenAI API key
// must be provided via the OPENAI_API_KEY environment variable.
func SendPrompt(modelID, prompt string) (string, error) {
	if strings.HasPrefix(modelID, "gpt-") {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return "", errors.New("OPENAI_API_KEY not set")
		}
		client := openai.NewClient(apiKey)
		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: modelID,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: prompt,
					},
				},
			},
		)
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", errors.New("empty response from model")
		}
		return resp.Choices[0].Message.Content, nil
	}
	return "", errors.New("unsupported model: " + modelID)
}
