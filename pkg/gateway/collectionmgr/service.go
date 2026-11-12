package collectionmgr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/modbusrtu"
	"harns-platform/pkg/gateway/collection/modbusrtuovertcp"
	"harns-platform/pkg/gateway/collection/modbustcp"
	"harns-platform/pkg/gateway/collection/opcua"
	"harns-platform/pkg/gateway/collection/s7"
	"harns-platform/pkg/gateway/collectionmgr/v1"
	"harns-platform/pkg/gateway/drivermgr"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/logging"
)

var ErrUnsupportedDriver = errors.New("unsupported driver")

// Service creates collection records.
type Service struct {
	store    *Store
	registry *createRegistry
	runtime  *RuntimeManager
}

// NewService creates a collection service with protocol create handlers.
func NewService(store *Store, handlers ...collection.CreateHandler) (*Service, error) {
	registry, err := newCreateRegistry(handlers...)
	if err != nil {
		return nil, err
	}
	return &Service{store: store, registry: registry}, nil
}

// SetRuntimeManager injects the collection runtime manager.
func (s *Service) SetRuntimeManager(runtime *RuntimeManager) {
	s.runtime = runtime
}

// StartRuntime loads persisted collections and starts the runtime manager.
func (s *Service) StartRuntime(ctx context.Context) error {
	if s.runtime == nil {
		return nil
	}
	records, err := s.store.ListAll(ctx)
	if err != nil {
		return response.ErrInternal(err)
	}
	enabled := make([]*CollectionRecord, 0, len(records))
	for _, record := range records {
		if record.Enable {
			enabled = append(enabled, record)
		}
	}
	if err := s.runtime.Load(ctx, enabled); err != nil {
		logging.ErrorS(err, "Failed to load collection runtime")
	}
	s.runtime.Start()
	return nil
}

// Close stops all collection collectors.
func (s *Service) Close(ctx context.Context) error {
	if s.runtime == nil {
		return nil
	}
	return s.runtime.Close(ctx)
}

type createEnvelope struct {
	DriverType  string `json:"driverType"`
	DriverModel string `json:"driverModel"`
}

// Create creates a collection from the unified /collections request body.
func (s *Service) Create(ctx context.Context, body []byte) (any, error) {
	var envelope createEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	envelope.DriverType = strings.TrimSpace(envelope.DriverType)
	envelope.DriverModel = strings.TrimSpace(envelope.DriverModel)
	issues := make([]validation.FieldIssue, 0, 2)
	if envelope.DriverType == "" {
		issues = append(issues, validation.FieldIssue{
			Field: "driverType",
			MsgID: "validate.collection_driver_type_required",
		})
	}
	if envelope.DriverModel == "" {
		issues = append(issues, validation.FieldIssue{
			Field: "driverModel",
			MsgID: "validate.collection_driver_model_required",
		})
	}
	if len(issues) > 0 {
		return nil, validation.New(issues)
	}

	exists, err := s.store.DriverExists(ctx, envelope.DriverType, envelope.DriverModel)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "driverType",
			MsgID: "validate.collection_driver_unsupported",
			Args:  []interface{}{envelope.DriverType, envelope.DriverModel},
		}})
	}
	rawSchema, err := s.store.DriverSchema(ctx, envelope.DriverType, envelope.DriverModel)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	schema, err := drivermgr.ParseSchema(rawSchema)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if err := schema.ValidateBody(body); err != nil {
		return nil, err
	}

	handler, ok := s.registry.resolve(envelope.DriverType, envelope.DriverModel)
	if !ok {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "driverType",
			MsgID: "validate.collection_driver_unsupported",
			Args:  []interface{}{envelope.DriverType, envelope.DriverModel},
		}})
	}
	request, err := handler.DecodeCreate(body)
	if err != nil {
		return nil, err
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	created, config, err := handler.BuildCreate(request)
	if err != nil {
		return nil, err
	}

	meta := created.GetCollectionMeta()
	now := time.Now().UTC()
	meta.ID = ulid.Make().String()
	meta.Version = "1"
	meta.ModTime = now
	record := &CollectionRecord{
		ID:               meta.ID,
		Name:             meta.Name,
		Version:          meta.Version,
		DriverType:       meta.DriverType,
		DriverModel:      meta.DriverModel,
		CollectStatus:    meta.CollectStatus,
		Enable:           true,
		CollectorCycleMs: meta.CollectorCycleMs,
		ProtocolConfig:   config,
		Variables:        meta.Variables,
		CreatedTime:      now,
		UpdatedTime:      now,
	}
	if err := s.store.Create(ctx, record); err != nil {
		return nil, err
	}
	if s.runtime != nil {
		if err := s.runtime.Register(ctx, record); err != nil {
			logging.ErrorS(err, "Failed to register collection runtime", "collectionId", record.ID)
		}
	}
	return s.collectionDetail(ctx, record)
}

// Get loads a collection record by ID.
func (s *Service) Get(ctx context.Context, id string) (*CollectionRecord, error) {
	return s.store.Get(ctx, id)
}

// List returns a page of collection summaries.
func (s *Service) List(ctx context.Context, q v1.CollectionQuery) (*apis.PageResp[*v1.CollectionVo], error) {
	defer logging.Trace("collectionmgr.List", "keyword", q.Keyword, "page", q.Page, "pageSize", q.PageSize)()

	q.Normalize()
	records, total, err := s.store.List(ctx, q.PageQuery, q.Keyword, q.DriverType, q.DriverModel)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	list := make([]*v1.CollectionVo, 0, len(records))
	for _, record := range records {
		if s.runtime != nil {
			if status, ok := s.runtime.Status(record.ID); ok {
				record.CollectStatus = status
			}
		}
		list = append(list, toCollectionVo(record))
	}
	return apis.NewPageResp(list, total, q.PageQuery), nil
}

// GetDetail returns protocol-specific collection detail without variables.
func (s *Service) GetDetail(ctx context.Context, id string) (any, error) {
	defer logging.Trace("collectionmgr.GetDetail", "id", id)()

	record, err := s.getCollection(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.collectionDetail(ctx, record)
}

// Update updates collection metadata and protocol config.
func (s *Service) Update(ctx context.Context, id string, body []byte) (any, error) {
	defer logging.Trace("collectionmgr.Update", "id", id)()

	record, err := s.getCollection(ctx, id)
	if err != nil {
		return nil, err
	}
	rawSchema, err := s.store.DriverSchema(ctx, record.DriverType, record.DriverModel)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	schema, err := drivermgr.ParseSchema(rawSchema)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if err := schema.ValidateBody(body); err != nil {
		return nil, err
	}

	var (
		name   string
		cycle  uint64
		config []byte
	)
	switch {
	case record.DriverType == modbustcp.DriverType && record.DriverModel == modbustcp.DriverModel:
		var req modbustcp.UpdateReq
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		item := &modbustcp.ModbusTcpCollection{}
		if err := item.ApplyConfig(record.ProtocolConfig); err != nil {
			return nil, response.ErrInternal(err)
		}
		item.ApplyUpdate(&req)
		name, cycle = item.Name, item.CollectorCycleMs
		config, err = item.ConfigJSON()
	case record.DriverType == modbusrtu.DriverType && record.DriverModel == modbusrtu.DriverModel:
		var req modbusrtu.UpdateReq
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		item := &modbusrtu.ModbusRtuCollection{}
		if err := item.ApplyConfig(record.ProtocolConfig); err != nil {
			return nil, response.ErrInternal(err)
		}
		item.ApplyUpdate(&req)
		name, cycle = item.Name, item.CollectorCycleMs
		config, err = item.ConfigJSON()
	case record.DriverType == modbusrtuovertcp.DriverType && record.DriverModel == modbusrtuovertcp.DriverModel:
		var req modbusrtuovertcp.UpdateReq
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		item := &modbusrtuovertcp.ModbusRtuOverTcpCollection{}
		if err := item.ApplyConfig(record.ProtocolConfig); err != nil {
			return nil, response.ErrInternal(err)
		}
		item.ApplyUpdate(&req)
		name, cycle = item.Name, item.CollectorCycleMs
		config, err = item.ConfigJSON()
	case record.DriverType == opcua.DriverType && record.DriverModel == opcua.DriverModel:
		var req opcua.UpdateReq
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		item := &opcua.OpcUaCollection{}
		if err := item.ApplyConfig(record.ProtocolConfig); err != nil {
			return nil, response.ErrInternal(err)
		}
		item.ApplyUpdate(&req)
		name, cycle = item.Name, item.CollectorCycleMs
		config, err = item.ConfigJSON()
	case record.DriverType == s7.DriverType &&
		(record.DriverModel == s7.DriverModel || record.DriverModel == s7.DriverModelS7200):
		var req s7.UpdateReq
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		item := &s7.S7Collection{}
		item.SetDriverModel(record.DriverModel)
		if err := item.ApplyConfig(record.ProtocolConfig); err != nil {
			return nil, response.ErrInternal(err)
		}
		item.ApplyUpdate(&req)
		name, cycle = item.Name, item.CollectorCycleMs
		config, err = item.ConfigJSON()
	default:
		return nil, ErrUnsupportedDriver
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if err := validateCommon(name, cycle); err != nil {
		return nil, err
	}
	record.Name = name
	record.CollectorCycleMs = cycle
	record.ProtocolConfig = config
	record.Version = nextVersion(record.Version)
	record.UpdatedTime = time.Now().UTC()
	if err := s.store.Update(ctx, record); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, response.ErrCollectionNotFound(id)
		}
		return nil, response.ErrInternal(err)
	}
	return s.collectionDetail(ctx, record)
}

// DeliverAction writes one or more collection variables.
func (s *Service) DeliverAction(ctx context.Context, id string, req v1.ActionReq) (*v1.ActionResp, error) {
	defer logging.Trace("collectionmgr.DeliverAction", "collectionId", id, "count", len(req.Actions))()

	record, err := s.getCollection(ctx, id)
	if err != nil {
		return nil, err
	}
	if !record.Enable {
		return nil, validation.New([]validation.FieldIssue{{Field: "enable", MsgID: "validate.collection_disabled"}})
	}
	if s.runtime == nil {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "collection", MsgID: "validate.action_collection_not_ready",
		}})
	}
	runtime, ok := s.runtime.Get(id)
	if !ok {
		return nil, response.ErrCollectionNotFound(id)
	}
	switch runtime.Collector.GetCollectStatus() {
	case collection.CollectStatusStopped, collection.CollectStatusUnconnected,
		collection.CollectStatusEmptyVariable, collection.CollectStatusError:
		return nil, validation.New([]validation.FieldIssue{{
			Field: "collection", MsgID: "validate.action_collection_not_ready",
		}})
	}
	variables := make(map[string]*collection.Variable)
	for _, variable := range runtime.Collector.GetCollectionMeta().SnapshotVariables() {
		if variable != nil {
			variables[variable.Name] = variable
		}
	}
	issues := make([]validation.FieldIssue, 0)
	seen := make(map[string]struct{}, len(req.Actions))
	actions := make([]collection.ActionValue, 0, len(req.Actions))
	for i, item := range req.Actions {
		name := strings.TrimSpace(item.Name)
		field := fmt.Sprintf("actions[%d].name", i)
		if name == "" {
			issues = append(issues, validation.FieldIssue{Field: field, MsgID: "validate.action_variable_required"})
			continue
		}
		if _, exists := seen[name]; exists {
			issues = append(issues, validation.FieldIssue{Field: field, MsgID: "validate.action_duplicate", Args: []interface{}{name}})
			continue
		}
		seen[name] = struct{}{}
		variable, exists := variables[name]
		if !exists {
			issues = append(issues, validation.FieldIssue{Field: field, MsgID: "validate.action_variable_not_found", Args: []interface{}{name}})
			continue
		}
		if variable.IsCalculated || (variable.AccessMode != collection.AccessModeWrite && variable.AccessMode != collection.AccessModeReadWrite) {
			issues = append(issues, validation.FieldIssue{Field: field, MsgID: "validate.action_variable_not_writable", Args: []interface{}{name}})
			continue
		}
		value, err := collection.DecodeActionValue(variable, item.Value)
		if err != nil {
			issues = append(issues, validation.FieldIssue{
				Field: fmt.Sprintf("actions[%d].value", i),
				MsgID: "validate.action_value_invalid",
				Args:  []interface{}{name, err.Error()},
			})
			continue
		}
		actions = append(actions, collection.ActionValue{Index: i, Variable: variable, Value: value})
	}
	if len(issues) > 0 {
		return nil, validation.New(issues)
	}
	if _, ok := runtime.Collector.(collection.ActionHandler); !ok {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "actions", MsgID: "validate.action_not_supported",
		}})
	}
	if err := runtime.DeliverAction(ctx, actions); err != nil {
		return nil, err
	}
	return &v1.ActionResp{Accepted: len(actions)}, nil
}

// Delete deletes one collection and its variables.
func (s *Service) Delete(ctx context.Context, id string) (int64, error) {
	defer logging.Trace("collectionmgr.Delete", "id", id)()

	if s.runtime != nil {
		if err := s.runtime.Remove(ctx, id); err != nil {
			logging.ErrorS(err, "Failed to remove collection runtime", "collectionId", id)
		}
	}
	deleted, err := s.store.Delete(ctx, id)
	if err != nil {
		return 0, response.ErrInternal(err)
	}
	if deleted == 0 {
		return 0, response.ErrCollectionNotFound(id)
	}
	return deleted, nil
}

// ListVariables returns a page of variables for one collection.
func (s *Service) ListVariables(ctx context.Context, collectionID string, q v1.VariableQuery) (*apis.PageResp[*v1.VariableVo], error) {
	defer logging.Trace("collectionmgr.ListVariables", "collectionId", collectionID, "page", q.Page, "pageSize", q.PageSize)()

	if _, err := s.getCollection(ctx, collectionID); err != nil {
		return nil, err
	}
	q.Normalize()
	variables, total, err := s.store.ListVariables(ctx, collectionID, q.PageQuery, q.Keyword)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if s.runtime != nil && len(variables) > 0 {
		names := make([]string, 0, len(variables))
		for _, variable := range variables {
			if variable != nil {
				names = append(names, variable.Name)
			}
		}
		values := s.runtime.SnapshotVariableValues(collectionID, names)
		for _, variable := range variables {
			if variable == nil {
				continue
			}
			if value, ok := values[variable.Name]; ok {
				variable.Value = value
			}
		}
	}
	list := make([]*v1.VariableVo, 0, len(variables))
	for _, variable := range variables {
		list = append(list, toVariableVo(variable))
	}
	return apis.NewPageResp(list, total, q.PageQuery), nil
}

// AddVariables appends variables to one collection atomically.
func (s *Service) AddVariables(ctx context.Context, collectionID string, variables []*collection.Variable) (*v1.AddVariablesVo, error) {
	defer logging.Trace("collectionmgr.AddVariables", "collectionId", collectionID, "count", len(variables))()

	record, err := s.getCollection(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	if len(variables) == 0 {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "variables",
			MsgID: "validate.collection_variables_required",
		}})
	}
	if err := collection.NormalizeVariables(variables); err != nil {
		return nil, err
	}
	rawSchema, err := s.store.DriverSchema(ctx, record.DriverType, record.DriverModel)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	schema, err := drivermgr.ParseSchema(rawSchema)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if err := schema.ValidateVariables(variables); err != nil {
		return nil, err
	}
	added, err := s.store.AddVariables(ctx, collectionID, variables)
	if err != nil {
		if errors.Is(err, ErrVariableConflict) {
			return nil, validation.New([]validation.FieldIssue{{
				Field: "variables",
				MsgID: "validate.collection_field_invalid",
				Args:  []interface{}{"variables"},
			}})
		}
		return nil, response.ErrInternal(err)
	}
	return &v1.AddVariablesVo{Added: int(added), Variables: variables}, nil
}

// DeleteVariables deletes variables by name from one collection.
func (s *Service) DeleteVariables(ctx context.Context, collectionID string, names []string) (int64, error) {
	defer logging.Trace("collectionmgr.DeleteVariables", "collectionId", collectionID, "count", len(names))()

	if _, err := s.getCollection(ctx, collectionID); err != nil {
		return 0, err
	}
	cleaned := dedupeStrings(names)
	deleted, err := s.store.DeleteVariables(ctx, collectionID, cleaned)
	if err != nil {
		return 0, response.ErrInternal(err)
	}
	return deleted, nil
}

// Stop pauses one collection's runtime without removing it.
func (s *Service) Stop(ctx context.Context, id string) (string, error) {
	defer logging.Trace("collectionmgr.Stop", "collectionId", id)()

	if _, err := s.getCollection(ctx, id); err != nil {
		return "", err
	}
	if s.runtime == nil {
		return "", response.ErrInternal(errors.New("collection runtime is not initialized"))
	}
	if err := s.runtime.StopCollection(ctx, id, "stopped manually"); err != nil {
		if errors.Is(err, ErrCollectionNotFound) {
			return collection.CollectStatusStopped, nil
		}
		return "", response.ErrInternal(err)
	}
	return collection.CollectStatusStopped, nil
}

// Start reloads one collection from storage and starts its runtime.
func (s *Service) Start(ctx context.Context, id string) (string, error) {
	defer logging.Trace("collectionmgr.Start", "collectionId", id)()

	record, err := s.store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", response.ErrCollectionNotFound(id)
	}
	if err != nil {
		return "", response.ErrInternal(err)
	}
	if !record.Enable {
		return "", validation.New([]validation.FieldIssue{{Field: "enable", MsgID: "validate.collection_disabled"}})
	}
	if s.runtime == nil {
		return "", response.ErrInternal(errors.New("collection runtime is not initialized"))
	}
	if current, ok := s.runtime.Get(id); ok && current.Collector.GetCollectStatus() == collection.CollectStatusCollecting {
		return collection.CollectStatusCollecting, nil
	}
	if err := s.runtime.Replace(ctx, record); err != nil {
		return "", response.ErrInternal(err)
	}
	status, _ := s.runtime.Status(id)
	if status == "" {
		status = collection.CollectStatusStopped
	}
	return status, nil
}

// SetEnable updates collection auto-start eligibility.
func (s *Service) SetEnable(ctx context.Context, id string, enable bool) (*v1.EnableVo, error) {
	defer logging.Trace("collectionmgr.SetEnable", "collectionId", id, "enable", enable)()

	record, err := s.getCollection(ctx, id)
	if err != nil {
		return nil, err
	}
	if !enable && s.runtime != nil {
		if runtime, ok := s.runtime.Get(id); ok && runtime.Collector.GetCollectStatus() != collection.CollectStatusStopped {
			return nil, validation.New([]validation.FieldIssue{{
				Field: "enable", MsgID: "validate.collection_disable_requires_stopped",
			}})
		}
	}
	record.Enable = enable
	record.Version = nextVersion(record.Version)
	record.UpdatedTime = time.Now().UTC()
	if err := s.store.Update(ctx, record); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, response.ErrCollectionNotFound(id)
		}
		return nil, response.ErrInternal(err)
	}
	return &v1.EnableVo{Enable: enable}, nil
}

// Restart reloads one collection from storage and replaces its runtime.
func (s *Service) Restart(ctx context.Context, id string) error {
	defer logging.Trace("collectionmgr.Restart", "collectionId", id)()

	if s.runtime == nil {
		return response.ErrInternal(errors.New("collection runtime is not initialized"))
	}
	record, err := s.store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return response.ErrCollectionNotFound(id)
	}
	if err != nil {
		return response.ErrInternal(err)
	}
	if !record.Enable {
		return validation.New([]validation.FieldIssue{{Field: "enable", MsgID: "validate.collection_disabled"}})
	}
	if err := s.runtime.Replace(ctx, record); err != nil {
		return response.ErrInternal(err)
	}
	return nil
}

func (s *Service) getCollection(ctx context.Context, id string) (*CollectionRecord, error) {
	record, err := s.store.GetCollection(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrCollectionNotFound(id)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	return record, nil
}

func toVariableVo(variable *collection.Variable) *v1.VariableVo {
	if variable == nil {
		return nil
	}
	return &v1.VariableVo{
		Name:         variable.Name,
		Type:         variable.Type,
		DataType:     variable.DataType,
		Address:      variable.Address,
		Bits:         variable.Bits,
		AccessMode:   variable.AccessMode,
		IsCalculated: variable.IsCalculated,
		Expression:   variable.Expression,
		Offset:       variable.Offset,
		Rate:         variable.Rate,
		Precision:    variable.Precision,
		Value:        variable.Value,
	}
}

func toCollectionVo(record *CollectionRecord) *v1.CollectionVo {
	return &v1.CollectionVo{
		ID:               record.ID,
		Name:             record.Name,
		Version:          record.Version,
		DriverType:       record.DriverType,
		DriverModel:      record.DriverModel,
		CollectStatus:    record.CollectStatus,
		Enable:           record.Enable,
		CollectorCycleMs: record.CollectorCycleMs,
		ModTime:          record.UpdatedTime,
	}
}

func (s *Service) collectionDetail(ctx context.Context, record *CollectionRecord) (*v1.CollectionDetailVo, error) {
	if s.runtime != nil {
		if status, ok := s.runtime.Status(record.ID); ok {
			record.CollectStatus = status
		}
	}
	vo := toCollectionVo(record)
	config := record.ProtocolConfig
	if handler, ok := s.registry.resolve(record.DriverType, record.DriverModel); ok {
		if redactor, ok := handler.(collection.ConfigRedactor); ok {
			config = redactor.RedactConfig(config)
		}
	}
	return &v1.CollectionDetailVo{
		CollectionVo:   *vo,
		ProtocolConfig: config,
	}, nil
}

func nextVersion(version string) string {
	n, err := strconv.ParseInt(version, 10, 64)
	if err != nil {
		return "1"
	}
	return strconv.FormatInt(n+1, 10)
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func validateCommon(name string, cycleMs uint64) error {
	issues := make([]validation.FieldIssue, 0, 2)
	if strings.TrimSpace(name) == "" {
		issues = append(issues, validation.FieldIssue{
			Field: "name",
			MsgID: "validate.collection_name_required",
		})
	}
	if cycleMs == 0 {
		issues = append(issues, validation.FieldIssue{
			Field: "collectorCycleMs",
			MsgID: "validate.collection_cycle_invalid",
		})
	}
	if len(issues) == 0 {
		return nil
	}
	return &validation.ValidationError{Issues: issues}
}
