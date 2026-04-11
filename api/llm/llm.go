package llm

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"google.golang.org/genai"
)

// Result holds the LLM response text and token usage.
type Result struct {
	Text        string
	TotalTokens int
}

var openAIModels = map[string]bool{
	"gpt-5.4": true, "gpt-5.4-mini": true, "gpt-5.4-nano": true,
}

var geminiModels = map[string]bool{
	"gemini-3.1-pro-preview": true, "gemini-3-flash-preview": true,
	"gemini-3.1-flash-lite-preview": true, "gemini-2.5-flash-lite": true,
}

type Client struct {
	OpenAIKey string
	GoogleKey string
}

func NewClient(openAIKey, googleKey string) *Client {
	return &Client{OpenAIKey: openAIKey, GoogleKey: googleKey}
}

func (c *Client) Call(ctx context.Context, model string, prompt string) (*Result, error) {
	if openAIModels[model] {
		return c.callOpenAI(ctx, model, prompt)
	}
	if geminiModels[model] {
		return c.callGemini(ctx, model, prompt)
	}
	return nil, fmt.Errorf("unsupported model: %s", model)
}

func (c *Client) callOpenAI(ctx context.Context, model string, prompt string) (*Result, error) {
	client := openai.NewClient(option.WithAPIKey(c.OpenAIKey))

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	totalTokens := int(resp.Usage.TotalTokens)

	return &Result{
		Text:        strings.TrimSpace(resp.Choices[0].Message.Content),
		TotalTokens: totalTokens,
	}, nil
}

func (c *Client) callGemini(ctx context.Context, model string, prompt string) (*Result, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: c.GoogleKey,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini client error: %w", err)
	}

	resp, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), nil)
	if err != nil {
		return nil, fmt.Errorf("gemini error: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no content")
	}

	var totalTokens int
	if resp.UsageMetadata != nil {
		totalTokens = int(resp.UsageMetadata.TotalTokenCount)
	}

	return &Result{
		Text:        strings.TrimSpace(resp.Text()),
		TotalTokens: totalTokens,
	}, nil
}
