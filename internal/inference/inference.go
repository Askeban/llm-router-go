package inference

import (
    "context"
    "errors"
    "os"
    "strings"

    anthropic "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
)

const defaultMaxTokens = 1024

// SendPrompt sends the given prompt to the specified model and returns the model's response.
//
// For model identifiers beginning with "claude-", the Anthropic API is used. The
// Anthropic client reads credentials from the ANTHROPIC_API_KEY environment
// variable and optionally the base URL from ANTHROPIC_API_URL.
func SendPrompt(modelID, prompt string) (string, error) {
    if strings.HasPrefix(modelID, "claude-") {
        return callAnthropic(modelID, prompt)
    }
    return "", errors.New("unsupported model")
}

func callAnthropic(modelID, prompt string) (string, error) {
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    if apiKey == "" {
        return "", errors.New("missing ANTHROPIC_API_KEY")
    }

    opts := []option.RequestOption{option.WithAPIKey(apiKey)}
    if base := os.Getenv("ANTHROPIC_API_URL"); base != "" {
        opts = append(opts, option.WithBaseURL(base))
    }

    client := anthropic.NewClient(opts...)
    msg, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
        Model:     anthropic.Model(modelID),
        MaxTokens: defaultMaxTokens,
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
        },
    })
    if err != nil {
        return "", err
    }

    var sb strings.Builder
    for _, block := range msg.Content {
        if block.Type == "text" {
            sb.WriteString(block.Text)
        }
    }
    return sb.String(), nil
}

