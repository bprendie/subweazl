package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bprendie/subweazl/internal/config"
)

type Client struct {
	cfg  config.LLMConfig
	http *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func New(cfg config.LLMConfig) Client {
	cfg.NormalizeForClient()
	return Client{cfg: cfg, http: http.DefaultClient}
}

func NewWithHTTP(cfg config.LLMConfig, httpClient *http.Client) Client {
	cfg.NormalizeForClient()
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return Client{cfg: cfg, http: httpClient}
}

func (c Client) Complete(ctx context.Context, messages []Message, maxTokens int) (string, error) {
	if c.cfg.BaseURL == "" || c.cfg.Model == "" || c.cfg.ChatPath == "" {
		return "", errors.New("llm is not configured")
	}
	body := map[string]any{
		"model":       c.cfg.Model,
		"messages":    messages,
		"temperature": 0.2,
	}
	if c.cfg.Provider == "ollama" {
		body["stream"] = false
		body["options"] = map[string]any{"temperature": 0.2, "num_predict": maxTokens}
		delete(body, "temperature")
	} else if maxTokens > 0 {
		body["max_tokens"] = maxTokens
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Text string `json:"text"`
		} `json:"choices"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Response string `json:"response"`
	}
	if err := c.post(ctx, c.cfg.ChatPath, body, &out); err != nil {
		return "", err
	}
	for _, choice := range out.Choices {
		if strings.TrimSpace(choice.Message.Content) != "" {
			return choice.Message.Content, nil
		}
		if strings.TrimSpace(choice.Text) != "" {
			return choice.Text, nil
		}
	}
	if strings.TrimSpace(out.Message.Content) != "" {
		return out.Message.Content, nil
	}
	if strings.TrimSpace(out.Response) != "" {
		return out.Response, nil
	}
	return "", errors.New("llm response contained no content")
}

// StreamComplete delivers text deltas as the provider produces them. vLLM
// uses OpenAI-compatible SSE; Ollama uses newline-delimited JSON.
func (c Client) StreamComplete(ctx context.Context, messages []Message, maxTokens int, onDelta func(string) error) error {
	if c.cfg.BaseURL == "" || c.cfg.Model == "" || c.cfg.ChatPath == "" {
		return errors.New("llm is not configured")
	}
	body := map[string]any{"model": c.cfg.Model, "messages": messages, "stream": true}
	if c.cfg.Provider == "ollama" {
		body["options"] = map[string]any{"temperature": 0.2, "num_predict": maxTokens}
	} else {
		body["temperature"] = 0.2
		if maxTokens > 0 {
			body["max_tokens"] = maxTokens
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(c.cfg.ChatPath), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return fmt.Errorf("llm http %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64<<10), 2<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "data: [DONE]" {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Response string `json:"response"`
			Error    string `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return fmt.Errorf("decode llm stream: %w", err)
		}
		if chunk.Error != "" {
			return errors.New(chunk.Error)
		}
		delta := chunk.Response
		if chunk.Message.Content != "" {
			delta = chunk.Message.Content
		}
		if len(chunk.Choices) > 0 {
			delta = chunk.Choices[0].Delta.Content
		}
		if delta != "" && onDelta != nil {
			if err := onDelta(delta); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func (c Client) FetchModels(ctx context.Context) ([]string, error) {
	if c.cfg.BaseURL == "" || c.cfg.ModelsPath == "" {
		return nil, errors.New("llm model listing is not configured")
	}
	var out struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
		Models []string `json:"models"`
	}
	if err := c.get(ctx, c.cfg.ModelsPath, &out); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var models []string
	for _, m := range out.Data {
		name := strings.TrimSpace(m.ID)
		if name == "" {
			name = strings.TrimSpace(m.Name)
		}
		if name != "" && !seen[name] {
			seen[name] = true
			models = append(models, name)
		}
	}
	for _, name := range out.Models {
		name = strings.TrimSpace(name)
		if name != "" && !seen[name] {
			seen[name] = true
			models = append(models, name)
		}
	}
	return models, nil
}

func (c Client) post(ctx context.Context, path string, body any, target any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(path), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	return c.do(req, target)
}

func (c Client) get(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path), nil)
	if err != nil {
		return err
	}
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	return c.do(req, target)
}

func (c Client) do(req *http.Request, target any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("llm http %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode llm response: %w", err)
	}
	return nil
}

func (c Client) url(path string) string {
	return strings.TrimRight(c.cfg.BaseURL, "/") + cleanPath(path)
}

func cleanPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
