package things

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"harns-platform/pkg/logging"
	"harns-platform/pkg/platform/thingmodel/tsl"
	"harns-platform/pkg/timeseries"
)

// PropertyProcessor handles decoded property reports.
type PropertyProcessor interface {
	ProcessProperty(ctx context.Context, msg *DeviceMessage, report *PropertyReport) error
}

// PropertyDefinitionResolver loads property definitions by thing model key.
type PropertyDefinitionResolver interface {
	GetPropertiesByKey(ctx context.Context, modelKey string) (map[string]tsl.Property, error)
}

// PropertyTimeseries writes property samples.
type PropertyTimeseries interface {
	WriteProperty(ctx context.Context, modelKey string, points []timeseries.PropertyPoint, debug bool) error
}

// PropertyReportLogger writes device and gateway property report logs.
type PropertyReportLogger interface {
	WritePropertyLog(ctx context.Context, modelKey string, log timeseries.PropertyLog) error
	WriteGatewayPropertyLog(ctx context.Context, log timeseries.GatewayPropertyLog) error
}

// ShadowUpdate is one property sample applied to the device shadow.
type ShadowUpdate struct {
	Time   time.Time
	Values map[string]any
}

// PropertyShadow merges reported property values into the device shadow.
type PropertyShadow interface {
	UpdateReported(ctx context.Context, thingID string, updates []ShadowUpdate) error
}

// LoggingPropertyProcessor is a placeholder that only logs the report.
type LoggingPropertyProcessor struct{}

func (p *LoggingPropertyProcessor) ProcessProperty(ctx context.Context, msg *DeviceMessage, report *PropertyReport) error {
	logging.V(2).InfoS("Received thing property report",
		"deviceKey", msg.DeviceKey, "topic", msg.Topic, "messageId", msg.MessageID,
		"source", msg.Source, "items", len(report.Payload))

	return nil
}

// DefaultPropertyProcessor validates property reports and submits samples to
// timeseries storage.
type DefaultPropertyProcessor struct {
	resolver        PropertyDefinitionResolver
	ts              PropertyTimeseries
	shadow          PropertyShadow
	trigger         PropertyTriggerSink
	gatewayResolver GatewayCollectionResolver
	snapshots       ThingSnapshotResolver
	logs            PropertyReportLogger
}

// NewPropertyProcessor builds the default property processor.
func NewPropertyProcessor(
	resolver PropertyDefinitionResolver,
	ts PropertyTimeseries,
	shadow PropertyShadow,
	trigger PropertyTriggerSink,
	gatewayResolver GatewayCollectionResolver,
	snapshots ThingSnapshotResolver,
	logs PropertyReportLogger,
) *DefaultPropertyProcessor {
	return &DefaultPropertyProcessor{
		resolver:        resolver,
		ts:              ts,
		shadow:          shadow,
		trigger:         trigger,
		gatewayResolver: gatewayResolver,
		snapshots:       snapshots,
		logs:            logs,
	}
}

func (p *DefaultPropertyProcessor) ProcessProperty(ctx context.Context, msg *DeviceMessage, report *PropertyReport) error {
	if p == nil || msg == nil {
		return nil
	}
	if report == nil {
		p.writeReportFailure(ctx, msg, errors.New("property report is nil"))
		return nil
	}
	if strings.TrimSpace(msg.ThingModelKey) != "" {
		return p.processDirect(ctx, msg, report)
	}
	if strings.TrimSpace(msg.GatewayID) != "" {
		return p.processGateway(ctx, msg, report)
	}
	logging.V(2).InfoS("Ignoring property report without thing model key or gateway id",
		"deviceKey", msg.DeviceKey, "messageId", msg.MessageID)
	return nil
}

func (p *DefaultPropertyProcessor) processDirect(ctx context.Context, msg *DeviceMessage, report *PropertyReport) error {
	modelKey := strings.TrimSpace(msg.ThingModelKey)
	debug := false
	if p.snapshots != nil && msg.ThingID != "" {
		snapshot, err := p.snapshots.GetThingSnapshot(ctx, msg.ThingID)
		if err != nil {
			logging.ErrorS(err, "Ignoring property report for unavailable thing",
				"thingId", msg.ThingID, "deviceKey", msg.DeviceKey, "messageId", msg.MessageID)
			return nil
		}
		debug = snapshot.Debug == 1
	}
	if p.resolver == nil {
		p.writeFailure(ctx, msg, modelKey, errors.New("property definition resolver is not configured"))
		return nil
	}

	definitions, err := p.resolver.GetPropertiesByKey(ctx, modelKey)
	if err != nil {
		p.writeFailure(ctx, msg, modelKey, fmt.Errorf("load thing model properties: %w", err))
		return nil
	}

	nowMillis := time.Now().UnixMilli()
	points := make([]timeseries.PropertyPoint, 0, len(report.Payload))
	shadowUpdates := make([]ShadowUpdate, 0, len(report.Payload))
	triggers := make([]PropertyTrigger, 0, len(report.Payload))
	for i, item := range report.Payload {
		values, err := p.convertPropertyValues(definitions, item.Property, nil)
		if err != nil {
			p.writeFailure(ctx, msg, modelKey, fmt.Errorf("payload[%d]: %w", i, err))
			return nil
		}

		timestamp := item.Time
		if timestamp <= 0 {
			timestamp = nowMillis
		}
		pointTime := time.UnixMilli(timestamp)
		points = append(points, timeseries.PropertyPoint{
			ModelID:   msg.ThingModelID,
			DeviceID:  msg.ThingID,
			DeviceKey: msg.DeviceKey,
			Time:      pointTime,
			Values:    values,
		})
		if len(values) > 0 {
			shadowUpdates = append(shadowUpdates, ShadowUpdate{Time: pointTime, Values: values})
			if !debug {
				triggers = append(triggers, PropertyTrigger{
					Time:       timestamp,
					DeviceID:   msg.ThingID,
					DeviceKey:  msg.DeviceKey,
					DeviceName: msg.DeviceName,
					ModelKey:   modelKey,
					Values:     values,
				})
			}
		}
	}

	if p.shadow != nil && len(shadowUpdates) > 0 {
		if msg.ThingID == "" {
			logging.V(2).InfoS("Cannot update thing shadow without thing id",
				"deviceKey", msg.DeviceKey, "messageId", msg.MessageID)
		} else if err := p.shadow.UpdateReported(ctx, msg.ThingID, shadowUpdates); err != nil {
			logging.ErrorS(err, "Failed to update thing shadow",
				"deviceKey", msg.DeviceKey, "thingId", msg.ThingID)
		}
	}

	if p.trigger != nil {
		for _, trigger := range triggers {
			p.trigger.Dispatch(ctx, trigger)
		}
	}

	if p.ts != nil {
		if err := p.ts.WriteProperty(ctx, modelKey, points, debug); err != nil && !errors.Is(err, timeseries.ErrDisabled) {
			p.writeFailure(ctx, msg, modelKey, fmt.Errorf("write property samples: %w", err))
			return nil
		}
	}

	if p.logs != nil {
		log := timeseries.PropertyLog{
			ModelID:   msg.ThingModelID,
			DeviceID:  msg.ThingID,
			DeviceKey: msg.DeviceKey,
			Time:      time.Now(),
			Success:   true,
			Payload:   msg.Payload,
		}
		if err := p.logs.WritePropertyLog(ctx, modelKey, log); err != nil && !errors.Is(err, timeseries.ErrDisabled) {
			logging.ErrorS(err, "Failed to write property log",
				"deviceKey", msg.DeviceKey, "thingModelKey", modelKey)
		}
	}
	return nil
}

func (p *DefaultPropertyProcessor) convertPropertyValues(
	definitions map[string]tsl.Property,
	raw map[string]json.RawMessage,
	mapping map[string]string,
) (map[string]any, error) {
	properties := raw
	if len(mapping) > 0 {
		properties = make(map[string]json.RawMessage, len(raw))
		for rawKey, value := range raw {
			identifier := strings.TrimSpace(mapping[rawKey])
			if identifier == "" {
				continue
			}
			if _, exists := properties[identifier]; exists {
				return nil, fmt.Errorf("duplicate mapping for property %s", identifier)
			}
			properties[identifier] = value
		}
	}

	for identifier, definition := range definitions {
		if !definition.Required {
			continue
		}
		value, ok := properties[identifier]
		if !ok || tsl.IsNullValue(value) {
			return nil, fmt.Errorf("required property is missing: %s", identifier)
		}
	}

	values := make(map[string]any, len(properties))
	for identifier, rawValue := range properties {
		definition, ok := definitions[identifier]
		if !ok {
			continue
		}
		if tsl.IsNullValue(rawValue) {
			continue
		}
		value, err := tsl.ConvertValue(definition.DataType, rawValue)
		if err != nil {
			return nil, fmt.Errorf("property %s: %w", identifier, err)
		}
		values[identifier] = value
	}
	return values, nil
}

func (p *DefaultPropertyProcessor) writeReportFailure(ctx context.Context, msg *DeviceMessage, cause error) {
	if msg == nil {
		return
	}
	if modelKey := strings.TrimSpace(msg.ThingModelKey); modelKey != "" {
		p.writeFailure(ctx, msg, modelKey, cause)
		return
	}
	if gatewayID := strings.TrimSpace(msg.GatewayID); gatewayID != "" {
		p.writeGatewayLog(ctx, msg, false, cause.Error())
		return
	}
	logging.V(2).InfoS("Property report validation failed",
		"deviceKey", msg.DeviceKey, "messageId", msg.MessageID, "err", cause)
}

func (p *DefaultPropertyProcessor) writeFailure(ctx context.Context, msg *DeviceMessage, modelKey string, cause error) {
	if msg == nil {
		return
	}
	if p == nil || p.logs == nil || modelKey == "" {
		logging.V(2).InfoS("Property report validation failed",
			"deviceKey", msg.DeviceKey, "messageId", msg.MessageID, "err", cause)
		return
	}
	log := timeseries.PropertyLog{
		ModelID:   msg.ThingModelID,
		DeviceID:  msg.ThingID,
		DeviceKey: msg.DeviceKey,
		Time:      time.Now(),
		Success:   false,
		Payload:   msg.Payload,
		ErrorMsg:  cause.Error(),
	}
	if err := p.logs.WritePropertyLog(ctx, modelKey, log); err != nil && !errors.Is(err, timeseries.ErrDisabled) {
		logging.ErrorS(err, "Failed to write property failure log",
			"deviceKey", msg.DeviceKey, "thingModelKey", modelKey)
	}
}
