package aimgr

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/aimgr/v1"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/logging"
)

// Service manages AI settings and the single MCP token.
type Service struct {
	store *Store
}

// NewService creates an AI configuration service.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// GetSettings returns AI settings.
func (s *Service) GetSettings(ctx context.Context) (*v1.SettingsVo, error) {
	defer logging.Trace("aimgr.GetSettings")()

	item, err := s.store.GetSettings(ctx)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	return toSettingsVo(item), nil
}

// RuntimeModel returns the enabled model configuration with its decrypted API key.
func (s *Service) RuntimeModel(ctx context.Context) (*RuntimeModel, error) {
	item, err := s.store.GetSettings(ctx)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if !item.AssistantEnabled {
		return nil, response.ErrAIProviderInvalid("assistant is disabled")
	}
	if item.APIKey == "" || item.EndpointURL == "" || item.Model == "" {
		return nil, response.ErrAIProviderInvalid("model configuration is incomplete")
	}
	return &RuntimeModel{
		Name:        item.Name,
		Remark:      item.Remark,
		Protocol:    ProtocolForEndpoint(item.EndpointURL),
		EndpointURL: ResolveEndpointURL(item.EndpointURL, ProtocolForEndpoint(item.EndpointURL)),
		Model:       item.Model,
		APIKey:      item.APIKey,
	}, nil
}

// UpdateSettings updates AI service switches without touching the model config.
func (s *Service) UpdateSettings(ctx context.Context, req *v1.UpdateSettingsReq) (*v1.SettingsVo, error) {
	defer logging.Trace("aimgr.UpdateSettings", "assistant", req.AssistantEnabled, "mcp", req.MCPEnabled)()

	item, err := s.store.GetSettings(ctx)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	item.AssistantEnabled = req.AssistantEnabled
	item.MCPEnabled = req.MCPEnabled
	switch {
	case item.MCPEnabled && item.MCPToken == "":
		value, err := newToken()
		if err != nil {
			return nil, response.ErrInternal(err)
		}
		item.MCPToken = value
	case !item.MCPEnabled:
		item.MCPToken = ""
	}
	if err := s.store.UpdateSettings(ctx, item); err != nil {
		return nil, response.ErrInternal(err)
	}
	return toSettingsVo(item), nil
}

// UpdateModel updates the single AI model configuration.
func (s *Service) UpdateModel(ctx context.Context, req *v1.UpdateModelReq) (*v1.SettingsVo, error) {
	defer logging.Trace("aimgr.UpdateModel", "name", req.Name, "endpointUrl", req.EndpointURL)()

	item, err := s.store.GetSettings(ctx)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		apiKey = item.APIKey
	}
	if err := validateModel(req.Name, req.Model, req.EndpointURL, apiKey); err != nil {
		return nil, err
	}
	item.Name = strings.TrimSpace(req.Name)
	item.Remark = strings.TrimSpace(req.Remark)
	item.Model = strings.TrimSpace(req.Model)
	item.EndpointURL = strings.TrimSpace(req.EndpointURL)
	item.APIKey = apiKey
	if err := s.store.UpdateSettings(ctx, item); err != nil {
		return nil, response.ErrInternal(err)
	}
	return toSettingsVo(item), nil
}

// TestModel verifies the model configuration. Empty fields use stored values.
func (s *Service) TestModel(ctx context.Context, req *v1.UpdateModelReq) error {
	defer logging.Trace("aimgr.TestModel")()

	item, err := s.store.GetSettings(ctx)
	if err != nil {
		return response.ErrInternal(err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = item.Name
	}
	endpointURL := strings.TrimSpace(req.EndpointURL)
	if endpointURL == "" {
		endpointURL = item.EndpointURL
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		apiKey = item.APIKey
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = item.Model
	}
	if err := validateModel(name, model, endpointURL, apiKey); err != nil {
		return err
	}
	if err := testProvider(ctx, ProtocolForEndpoint(endpointURL), endpointURL, model, apiKey); err != nil {
		return response.ErrAIProviderInvalid(err.Error())
	}
	return nil
}

func validateModel(name, model, endpointURL, apiKey string) error {
	issues := make([]validation.FieldIssue, 0, 4)
	name = strings.TrimSpace(name)
	model = strings.TrimSpace(model)
	endpointURL = strings.TrimSpace(endpointURL)
	apiKey = strings.TrimSpace(apiKey)
	parsed, err := url.Parse(endpointURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		issues = append(issues, validation.FieldIssue{Field: "endpointUrl", MsgID: "validate.ai_provider_endpoint_invalid"})
	}
	if name == "" {
		issues = append(issues, validation.FieldIssue{Field: "name", MsgID: "validate.ai_provider_name_required"})
	}
	if model == "" {
		issues = append(issues, validation.FieldIssue{Field: "model", MsgID: "validate.ai_provider_model_required"})
	}
	if apiKey == "" {
		issues = append(issues, validation.FieldIssue{Field: "apiKey", MsgID: "validate.ai_provider_api_key_required"})
	}
	if len(issues) == 0 {
		return nil
	}
	return &validation.ValidationError{Issues: issues}
}

func ProtocolForEndpoint(endpointURL string) string {
	lower := strings.ToLower(endpointURL)
	if strings.Contains(lower, "anthropic") || strings.Contains(lower, "/messages") {
		return ProtocolAnthropicMessages
	}
	return ProtocolOpenAICompatible
}

func maskedAPIKey(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "********"
}

func newToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate mcp token: %w", err)
	}
	return "mcp_" + base64.RawURLEncoding.EncodeToString(buffer), nil
}

func toSettingsVo(item *Settings) *v1.SettingsVo {
	if item == nil {
		return nil
	}
	return &v1.SettingsVo{
		AssistantEnabled: item.AssistantEnabled,
		MCPEnabled:       item.MCPEnabled,
		Name:             item.Name,
		Remark:           item.Remark,
		Model:            item.Model,
		EndpointURL:      item.EndpointURL,
		APIKey:           maskedAPIKey(item.APIKey),
		HasAPIKey:        item.APIKey != "",
		MCPToken:         item.MCPToken,
		UpdatedTime:      item.UpdatedTime,
	}
}
