package emqx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// PublishRequest is the EMQX v5 publish payload.
type PublishRequest struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
	QoS     int    `json:"qos"`
}

// Service publishes MQTT messages through the EMQX HTTP API.
type Service struct {
	cfg    Config
	client *http.Client
}

// NewService builds an EMQX API client.
func NewService(cfg Config) *Service {
	return &Service{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout()},
	}
}

// Publish sends a message to EMQX via POST /api/v5/publish.
func (s *Service) Publish(ctx context.Context, req PublishRequest) error {
	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		return fmt.Errorf("emqx: topic is required")
	}
	if req.QoS < 0 || req.QoS > 2 {
		return fmt.Errorf("emqx: invalid qos %d", req.QoS)
	}
	if strings.TrimSpace(s.cfg.APIURL) == "" {
		return fmt.Errorf("emqx: api-url is empty")
	}

	body, err := json.Marshal(PublishRequest{Topic: topic, Payload: req.Payload, QoS: req.QoS})
	if err != nil {
		return err
	}

	endpoint := strings.TrimRight(s.cfg.APIURL, "/") + "/api/v5/publish"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(s.cfg.AK, s.cfg.SK)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("emqx: publish failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("emqx: publish failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(payload)))
}
