package llm

import (
	"context"
	"refactai/internal/config"

	"google.golang.org/genai"
)

type Client struct {
	client *genai.Client
	model  string
}

func NewGemini(ctx context.Context, cfg *config.Config) (*Client, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: cfg.GeminiApiKey,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
		model:  cfg.GeminiModel,
	}, nil
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.Models.GenerateContent(
		ctx,
		c.model,
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		return "", err
	}

	return resp.Text(), nil
}
