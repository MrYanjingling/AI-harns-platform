package drivermgr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/drivermgr/v1"
	"harns-platform/pkg/logging"
)

var pluginNamePattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

// Service manages driver metadata.
type Service struct {
	store *Store
}

// NewService creates a driver metadata service.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// List returns all drivers.
func (s *Service) List(ctx context.Context, driverType string) ([]*v1.DriverVo, error) {
	defer logging.Trace("drivermgr.List", "driverType", driverType)()

	items, err := s.store.List(ctx, driverType)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	out := make([]*v1.DriverVo, 0, len(items))
	for _, item := range items {
		out = append(out, toVo(item))
	}
	return out, nil
}

// ListGrouped returns drivers grouped by driver type.
func (s *Service) ListGrouped(ctx context.Context, driverType string) ([]*v1.DriverGroupVo, error) {
	defer logging.Trace("drivermgr.ListGrouped", "driverType", driverType)()

	items, err := s.store.List(ctx, driverType)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	groups := make([]*v1.DriverGroupVo, 0)
	index := make(map[string]int)
	for _, item := range items {
		vo := toVo(item)
		if groupIndex, ok := index[vo.DriverType]; ok {
			groups[groupIndex].Models = append(groups[groupIndex].Models, vo)
			continue
		}
		index[vo.DriverType] = len(groups)
		groups = append(groups, &v1.DriverGroupVo{
			DriverType: vo.DriverType,
			Models:     []*v1.DriverVo{vo},
		})
	}
	return groups, nil
}

// Get returns one driver.
func (s *Service) Get(ctx context.Context, id string) (*v1.DriverVo, error) {
	defer logging.Trace("drivermgr.Get", "id", id)()

	item, err := s.store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrDriverNotFound(id)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	return toVo(item), nil
}

// GetSchemaByModel returns the parsed schema for one driver model.
func (s *Service) GetSchemaByModel(ctx context.Context, model string) (*v1.DriverSchemaVo, error) {
	defer logging.Trace("drivermgr.GetSchemaByModel", "model", model)()

	model = strings.TrimSpace(model)
	if model == "" {
		return nil, &ValidationError{Issues: []FieldIssue{{
			Field: "driverModel",
			MsgID: "validate.driver_model_required",
		}}}
	}
	item, err := s.store.GetByModel(ctx, model)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrDriverNotFound(model)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if _, err := ParseSchema(item.Schema); err != nil {
		return nil, response.ErrDriverInvalid(err.Error())
	}
	return &v1.DriverSchemaVo{
		DriverType:  item.DriverType,
		DriverModel: item.Model,
		Schema:      item.Schema,
	}, nil
}

// Create creates a non-built-in driver.
func (s *Service) Create(ctx context.Context, req *v1.CreateDriverReq) (*v1.DriverVo, error) {
	defer logging.Trace("drivermgr.Create", "driverType", req.DriverType, "model", req.Model)()

	if err := validateDriver(req.DriverType, req.Model, req.PluginName, req.Schema); err != nil {
		return nil, err
	}
	item := &Driver{
		ID:            ulid.Make().String(),
		DriverType:    strings.TrimSpace(req.DriverType),
		Model:         strings.TrimSpace(req.Model),
		BuiltinDriver: false,
		Schema:        normalizeSchema(req.Schema),
		PluginName:    strings.TrimSpace(req.PluginName),
		CreatedTime:   time.Now().UTC(),
	}
	if err := s.store.Create(ctx, item); err != nil {
		return nil, mapServiceStoreError(err, item.PluginName)
	}
	return toVo(item), nil
}

// Update updates a custom driver.
func (s *Service) Update(ctx context.Context, id string, req *v1.UpdateDriverReq) (*v1.DriverVo, error) {
	defer logging.Trace("drivermgr.Update", "id", id, "driverType", req.DriverType, "model", req.Model)()

	item, err := s.store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrDriverNotFound(id)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if item.BuiltinDriver {
		return nil, response.ErrDriverBuiltinReadOnly()
	}
	if err := validateDriver(req.DriverType, req.Model, req.PluginName, req.Schema); err != nil {
		return nil, err
	}
	item.DriverType = strings.TrimSpace(req.DriverType)
	item.Model = strings.TrimSpace(req.Model)
	item.Schema = normalizeSchema(req.Schema)
	item.PluginName = strings.TrimSpace(req.PluginName)
	if err := s.store.Update(ctx, item); err != nil {
		return nil, mapServiceStoreError(err, item.PluginName)
	}
	return toVo(item), nil
}

// Delete removes a custom driver.
func (s *Service) Delete(ctx context.Context, id string) error {
	defer logging.Trace("drivermgr.Delete", "id", id)()

	item, err := s.store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return response.ErrDriverNotFound(id)
	}
	if err != nil {
		return response.ErrInternal(err)
	}
	if item.BuiltinDriver {
		return response.ErrDriverBuiltinReadOnly()
	}
	if err := s.store.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrBuiltinReadOnly) {
			return response.ErrDriverBuiltinReadOnly()
		}
		return response.ErrInternal(err)
	}
	return nil
}

func validateDriver(driverType, model, pluginName string, schema json.RawMessage) error {
	issues := make([]FieldIssue, 0, 4)
	if strings.TrimSpace(driverType) == "" {
		issues = append(issues, FieldIssue{Field: "driverType", MsgID: "validate.driver_type_required"})
	}
	if strings.TrimSpace(model) == "" {
		issues = append(issues, FieldIssue{Field: "model", MsgID: "validate.driver_model_required"})
	}
	if strings.TrimSpace(pluginName) == "" {
		issues = append(issues, FieldIssue{Field: "pluginName", MsgID: "validate.driver_plugin_name_required"})
	} else if !pluginNamePattern.MatchString(strings.TrimSpace(pluginName)) {
		issues = append(issues, FieldIssue{Field: "pluginName", MsgID: "validate.driver_plugin_name_invalid"})
	}
	if !validSchema(schema) {
		issues = append(issues, FieldIssue{Field: "schema", MsgID: "validate.driver_schema_invalid"})
	}
	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

func validSchema(schema json.RawMessage) bool {
	_, err := ParseSchema(schema)
	return err == nil
}

func normalizeSchema(schema json.RawMessage) json.RawMessage {
	if len(schema) == 0 {
		return json.RawMessage(`{"fields":[]}`)
	}
	return schema
}

func mapServiceStoreError(err error, value string) error {
	switch {
	case errors.Is(err, ErrConflict):
		return response.ErrDriverConflict(value)
	case errors.Is(err, ErrBuiltinReadOnly):
		return response.ErrDriverBuiltinReadOnly()
	default:
		return response.ErrInternal(err)
	}
}

func toVo(item *Driver) *v1.DriverVo {
	return &v1.DriverVo{
		ID:            item.ID,
		DriverType:    item.DriverType,
		Model:         item.Model,
		BuiltinDriver: item.BuiltinDriver,
		Schema:        item.Schema,
		PluginName:    item.PluginName,
		CreatedTime:   item.CreatedTime,
	}
}
