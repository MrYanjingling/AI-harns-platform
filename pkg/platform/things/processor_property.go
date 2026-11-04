package things

import (
	"context"
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

// PropertyTimeseries writes property samples and request logs.
type PropertyTimeseries interface {
	WriteProperty(ctx context.Context, modelKey string, points []timeseries.PropertyPoint) error
	WritePropertyLog(ctx context.Context, modelKey string, log timeseries.PropertyLog) error
}

// PropertyLogWriter writes property request logs.
type PropertyLogWriter interface {
	WritePropertyLog(ctx context.Context, modelKey string, log timeseries.PropertyLog) error
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
	resolver PropertyDefinitionResolver
	ts       PropertyTimeseries
	shadow   PropertyShadow
	trigger  PropertyTriggerSink
}

// NewPropertyProcessor builds the default property processor.
func NewPropertyProcessor(resolver PropertyDefinitionResolver, ts PropertyTimeseries, shadow PropertyShadow, trigger PropertyTriggerSink) *DefaultPropertyProcessor {
	return &DefaultPropertyProcessor{resolver: resolver, ts: ts, shadow: shadow, trigger: trigger}
}

func (p *DefaultPropertyProcessor) ProcessProperty(ctx context.Context, msg *DeviceMessage, report *PropertyReport) error {
	if p == nil || msg == nil {
		return nil
	}
	modelKey := strings.TrimSpace(msg.ThingModelKey)
	if report == nil {
		p.writeFailure(ctx, msg, modelKey, errors.New("property report is nil"))
		return nil
	}
	if p.resolver == nil {
		p.writeFailure(ctx, msg, modelKey, errors.New("property definition resolver is not configured"))
		return nil
	}
	if modelKey == "" {
		logging.V(2).InfoS("Ignoring property report without thing model key",
			"deviceKey", msg.DeviceKey, "messageId", msg.MessageID)
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
		for identifier, definition := range definitions {
			if !definition.Required {
				continue
			}
			raw, ok := item.Property[identifier]
			if !ok || tsl.IsNullValue(raw) {
				p.writeFailure(ctx, msg, modelKey,
					fmt.Errorf("payload[%d].property.%s: required property is missing", i, identifier))
				return nil
			}
		}

		values := make(map[string]any, len(item.Property))
		for identifier, raw := range item.Property {
			definition, ok := definitions[identifier]
			if !ok {
				continue
			}
			if tsl.IsNullValue(raw) {
				continue
			}
			value, err := tsl.ConvertValue(definition.DataType, raw)
			if err != nil {
				p.writeFailure(ctx, msg, modelKey,
					fmt.Errorf("payload[%d].property.%s: %w", i, identifier, err))
				return nil
			}
			values[identifier] = value
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
		if err := p.ts.WriteProperty(ctx, modelKey, points); err != nil && !errors.Is(err, timeseries.ErrDisabled) {
			p.writeFailure(ctx, msg, modelKey, fmt.Errorf("write property samples: %w", err))
			return nil
		}
	}

	if p.ts != nil {
		log := timeseries.PropertyLog{
			ModelID:   msg.ThingModelID,
			DeviceID:  msg.ThingID,
			DeviceKey: msg.DeviceKey,
			Time:      time.Now(),
			Success:   true,
			Payload:   msg.Payload,
		}
		if err := p.ts.WritePropertyLog(ctx, modelKey, log); err != nil && !errors.Is(err, timeseries.ErrDisabled) {
			logging.ErrorS(err, "Failed to write property log",
				"deviceKey", msg.DeviceKey, "thingModelKey", modelKey)
		}
	}
	return nil
}

func (p *DefaultPropertyProcessor) writeFailure(ctx context.Context, msg *DeviceMessage, modelKey string, cause error) {
	if msg == nil {
		return
	}
	if p == nil || p.ts == nil || modelKey == "" {
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
	if err := p.ts.WritePropertyLog(ctx, modelKey, log); err != nil && !errors.Is(err, timeseries.ErrDisabled) {
		logging.ErrorS(err, "Failed to write property failure log",
			"deviceKey", msg.DeviceKey, "thingModelKey", modelKey)
	}
}
