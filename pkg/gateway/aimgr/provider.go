package aimgr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func testProvider(ctx context.Context, protocol, endpointURL, model, apiKey string) error {
	if strings.TrimSpace(endpointURL) == "" {
		return fmt.Errorf("endpoint URL is required")
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("API key is required")
	}
	endpointURL = ResolveEndpointURL(endpointURL, protocol)
	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{{
			"role":    "user",
			"content": "ping",
		}},
	}
	headers := map[string]string{"Content-Type": "application/json"}
	switch protocol {
	case ProtocolOpenAICompatible:
		payload["max_tokens"] = 1
		payload["stream"] = false
		headers["Authorization"] = "Bearer " + apiKey
	case ProtocolAnthropicMessages:
		payload["max_tokens"] = 1
		headers["x-api-key"] = apiKey
		headers["anthropic-version"] = "2023-06-01"
	default:
		return fmt.Errorf("unsupported provider protocol %q", protocol)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode provider request: %w", err)
	}
	req, err := http.NewRequestWithContext(testCtx, http.MethodPost, endpointURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create provider request: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("request provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		message := strings.TrimSpace(string(raw))
		if message == "" {
			message = resp.Status
		}
		return fmt.Errorf("provider returned HTTP %d at %s: %s", resp.StatusCode, displayEndpoint(endpointURL), message)
	}
	return nil
}

func ResolveEndpointURL(endpointURL, protocol string) string {
	parsed, err := url.Parse(strings.TrimSpace(endpointURL))
	if err != nil {
		return endpointURL
	}
	path := strings.TrimRight(parsed.Path, "/")
	if protocol == ProtocolAnthropicMessages {
		switch {
		case strings.HasSuffix(path, "/messages"):
			parsed.Path = path
		case path == "" || path == "/":
			parsed.Path = "/v1/messages"
		case strings.HasSuffix(path, "/v1"):
			parsed.Path = path + "/messages"
		default:
			parsed.Path = path + "/messages"
		}
		return parsed.String()
	}
	switch {
	case strings.HasSuffix(path, "/chat/completions"):
		parsed.Path = path
	case path == "" || path == "/":
		parsed.Path = "/v1/chat/completions"
	case strings.HasSuffix(path, "/v1"):
		parsed.Path = path + "/chat/completions"
	default:
		parsed.Path = path + "/chat/completions"
	}
	return parsed.String()
}

func displayEndpoint(endpointURL string) string {
	parsed, err := url.Parse(endpointURL)
	if err != nil {
		return endpointURL
	}
	parsed.RawQuery = ""
	return parsed.String()
}
