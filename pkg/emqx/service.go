package emqx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
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

// Broker returns the configured MQTT broker address.
func (s *Service) Broker() string {
	if s == nil {
		return ""
	}
	return s.cfg.Broker
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

// KickClient disconnects one MQTT client through the EMQX HTTP API.
func (s *Service) KickClient(ctx context.Context, clientID string) error {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return fmt.Errorf("emqx: client id is required")
	}
	if s == nil {
		return nil
	}
	if strings.TrimSpace(s.cfg.APIURL) == "" {
		return fmt.Errorf("emqx: api-url is empty")
	}

	endpoint := strings.TrimRight(s.cfg.APIURL, "/") + "/api/v5/clients/" + url.PathEscape(clientID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	httpReq.SetBasicAuth(s.cfg.AK, s.cfg.SK)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("emqx: kick client failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("emqx: kick client failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(payload)))
}

// KickClients disconnects multiple MQTT clients, preferring the EMQX bulk API.
func (s *Service) KickClients(ctx context.Context, clientIDs []string) error {
	seen := make(map[string]struct{}, len(clientIDs))
	ids := make([]string, 0, len(clientIDs))
	for _, clientID := range clientIDs {
		clientID = strings.TrimSpace(clientID)
		if clientID == "" {
			continue
		}
		if _, ok := seen[clientID]; ok {
			continue
		}
		seen[clientID] = struct{}{}
		ids = append(ids, clientID)
	}
	if len(ids) == 0 {
		return nil
	}

	handled, err := s.kickClientsBulk(ctx, ids)
	if handled {
		return err
	}
	return s.kickClientsIndividually(ctx, ids)
}

func (s *Service) kickClientsBulk(ctx context.Context, clientIDs []string) (bool, error) {
	if s == nil {
		return true, nil
	}
	if strings.TrimSpace(s.cfg.APIURL) == "" {
		return true, fmt.Errorf("emqx: api-url is empty")
	}
	body, err := json.Marshal(clientIDs)
	if err != nil {
		return true, err
	}

	endpoint := strings.TrimRight(s.cfg.APIURL, "/") + "/api/v5/clients/kickout/bulk"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return true, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(s.cfg.AK, s.cfg.SK)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return true, fmt.Errorf("emqx: bulk kick clients failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return false, nil
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return true, fmt.Errorf("emqx: bulk kick clients failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(payload)))
}

func (s *Service) kickClientsIndividually(ctx context.Context, clientIDs []string) error {
	sem := make(chan struct{}, 5)
	errCh := make(chan error, len(clientIDs))
	var wg sync.WaitGroup
	for _, clientID := range clientIDs {
		clientID := clientID
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
			if err := s.KickClient(ctx, clientID); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)

	errs := make([]error, 0)
	for err := range errCh {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
