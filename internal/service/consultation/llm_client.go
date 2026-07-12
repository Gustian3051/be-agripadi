package consultation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gustian305/backend/logger"
)

type OpenAICompatibleClient struct {
	apiURL     string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewOpenAICompatibleClient(
	apiURL string,
	apiKey string,
	model string,
	timeout time.Duration,
) *OpenAICompatibleClient {

	apiURL = strings.TrimRight(
		strings.TrimSpace(apiURL),
		"/",
	)

	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)

	if timeout <= 0 {
		timeout = DefaultLLMTimeout
	}

	if apiURL == "" ||
		apiKey == "" ||
		model == "" {

		return nil
	}

	return &OpenAICompatibleClient{
		apiURL: apiURL,
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type chatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []chatCompletionMessage `json:"messages"`
	Temperature float64                 `json:"temperature"`
	TopP        float64                 `json:"top_p,omitempty"`
	MaxTokens   int                     `json:"max_tokens"`
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message      chatCompletionMessage `json:"message"`
		FinishReason string                `json:"finish_reason"`
	} `json:"choices"`

	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *OpenAICompatibleClient) Generate(ctx context.Context, prompt string) (string, error) {
	startedAt := time.Now()
	operation := "Generate"

	model := ""
	if c != nil {
		model = c.model
	}

	logger.Request(
		"llm.client",
		operation,
		slog.String("model", model),
		slog.Int("prompt_length", len(prompt)),
	)
	logger.DebugPayload(
		"llm.client",
		operation,
		slog.String("prompt", logger.Truncate(prompt, 1200)),
	)

	if c == nil ||
		c.httpClient == nil ||
		c.apiURL == "" ||
		c.apiKey == "" ||
		c.model == "" {

		err := errors.New(
			"llm client is not configured",
		)
		logger.Failure("llm.client", operation, startedAt, err)
		return "", err
	}

	body := chatCompletionRequest{
		Model: c.model,
		Messages: []chatCompletionMessage{
			{
				Role: "system",
				Content: `
Anda adalah AI Agronomist profesional untuk budidaya padi.

Tugas Anda adalah membantu petani memahami hasil diagnosis hama dan rekomendasi pengendalian, serta konsultasi tentang pertanian.

Berikan jawaban yang:
- faktual
- singkat
- tidak mengarang data
- tidak mengubah hasil diagnosis sistem
- tidak mengubah rekomendasi pestisida sistem
`,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.2,
		TopP:        0.8,
		MaxTokens:   2048,
	}

	payload, err := json.Marshal(
		body,
	)

	if err != nil {
		logger.Failure("llm.client", operation, startedAt, err)
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.apiURL+"/chat/completions",
		bytes.NewReader(payload),
	)

	if err != nil {
		logger.Failure("llm.client", operation, startedAt, err)
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(
		req,
	)

	if err != nil {
		logger.Failure("llm.client", operation, startedAt, err)
		return "", err
	}

	defer resp.Body.Close()

	var result chatCompletionResponse

	if err := json.NewDecoder(resp.Body).Decode(
		&result,
	); err != nil {

		logger.Failure(
			"llm.client",
			operation,
			startedAt,
			err,
			slog.Int("status_code", resp.StatusCode),
		)
		return "", err
	}

	if resp.StatusCode < 200 ||
		resp.StatusCode >= 300 {

		if result.Error != nil &&
			result.Error.Message != "" {

			err := errors.New(
				result.Error.Message,
			)
			logger.Failure(
				"llm.client",
				operation,
				startedAt,
				err,
				slog.Int("status_code", resp.StatusCode),
			)
			return "", err
		}

		err := errors.New(
			resp.Status,
		)
		logger.Failure(
			"llm.client",
			operation,
			startedAt,
			err,
			slog.Int("status_code", resp.StatusCode),
		)

		finishReason := ""

		if len(result.Choices) > 0 {
			finishReason =
				result.Choices[0].FinishReason
		}

		logger.Response(
			"llm.client",
			operation,
			startedAt,
			slog.String(
				"finish_reason",
				finishReason,
			),
		)

		return "", err
	}

	if len(result.Choices) == 0 ||
		strings.TrimSpace(result.Choices[0].Message.Content) == "" {

		err := errors.New(
			"empty llm response",
		)
		logger.Failure(
			"llm.client",
			operation,
			startedAt,
			err,
			slog.Int("status_code", resp.StatusCode),
		)
		return "", err
	}

	content := result.Choices[0].Message.Content

	finishReason := ""

	if len(result.Choices) > 0 {
		finishReason =
			result.Choices[0].FinishReason
	}

	logger.Response(
		"llm.client",
		operation,
		startedAt,
		slog.String("model", c.model),
		slog.Int("status_code", resp.StatusCode),
		slog.Int("response_length", len(content)),
		slog.String("finish_reason", finishReason),
	)

	if len(result.Choices) > 0 {
		finishReason =
			result.Choices[0].FinishReason
	}

	logger.Response(
		"llm.client",
		operation,
		startedAt,
		slog.String(
			"finish_reason",
			finishReason,
		),
	)

	return content, nil
}
