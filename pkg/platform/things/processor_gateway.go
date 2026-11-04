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

func (p *DefaultPropertyProcessor) processGateway(ctx context.Context, msg *DeviceMessage, report *PropertyReport) error {
	if p.gatewayResolver == nil {
		p.writeGatewayLog(ctx, msg, false, "gateway collection resolver is not configured")
		return nil
	}

	nowMillis := time.Now().UnixMilli()
	pointsByModel := make(map[string][]timeseries.PropertyPoint)
	shadowByThing := make(map[string][]ShadowUpdate)
	triggers := make([]PropertyTrigger, 0, len(report.Payload))
	definitions := make(map[string]map[string]tsl.Property)
	issues := make([]string, 0)

	for i, item := range report.Payload {
		collectionID := strings.TrimSpace(item.CollectionID)
		if collectionID == "" {
			issues = append(issues, fmt.Sprintf("payload[%d]: collectionId is required", i))
			continue
		}

		bindings, err := p.gatewayResolver.Resolve(ctx, msg.GatewayType, msg.GatewayID, collectionID)
		if err != nil {
			issues = append(issues, fmt.Sprintf("payload[%d].collectionId=%s: %v", i, collectionID, err))
			continue
		}
		if len(bindings) == 0 {
			issues = append(issues, fmt.Sprintf("payload[%d].collectionId=%s: no device binding", i, collectionID))
			continue
		}

		timestamp := item.Time
		if timestamp <= 0 {
			timestamp = nowMillis
		}
		pointTime := time.UnixMilli(timestamp)

		for _, binding := range bindings {
			thingID := strings.TrimSpace(binding.ThingID)
			modelKey := strings.TrimSpace(binding.ThingModelKey)
			if thingID == "" || modelKey == "" {
				issues = append(issues, fmt.Sprintf("payload[%d].collectionId=%s: invalid binding without thingId or modelKey", i, collectionID))
				continue
			}

			modelDefinitions, err := p.gatewayDefinitions(ctx, modelKey, definitions)
			if err != nil {
				issues = append(issues, fmt.Sprintf("payload[%d].collectionId=%s.thingId=%s: load model properties: %v", i, collectionID, thingID, err))
				continue
			}
			values, err := p.convertPropertyValues(modelDefinitions, item.Property, binding.PropertyMapping)
			if err != nil {
				issues = append(issues, fmt.Sprintf("payload[%d].collectionId=%s.thingId=%s: %v", i, collectionID, thingID, err))
				continue
			}
			if len(values) == 0 {
				continue
			}

			pointsByModel[modelKey] = append(pointsByModel[modelKey], timeseries.PropertyPoint{
				ModelID:   binding.ThingModelID,
				DeviceID:  thingID,
				DeviceKey: binding.DeviceKey,
				Time:      pointTime,
				Values:    values,
			})
			shadowByThing[thingID] = append(shadowByThing[thingID], ShadowUpdate{Time: pointTime, Values: values})
			triggers = append(triggers, PropertyTrigger{
				Time:       timestamp,
				DeviceID:   thingID,
				DeviceKey:  binding.DeviceKey,
				DeviceName: binding.DeviceName,
				ModelKey:   modelKey,
				Values:     values,
			})
		}
	}

	if p.shadow != nil {
		for thingID, updates := range shadowByThing {
			if err := p.shadow.UpdateReported(ctx, thingID, updates); err != nil {
				logging.ErrorS(err, "Failed to update gateway thing shadow", "thingId", thingID)
			}
		}
	}

	if p.trigger != nil {
		for _, trigger := range triggers {
			p.trigger.Dispatch(ctx, trigger)
		}
	}

	if p.ts != nil {
		for modelKey, points := range pointsByModel {
			if err := p.ts.WriteProperty(ctx, modelKey, points); err != nil && !errors.Is(err, timeseries.ErrDisabled) {
				issues = append(issues, fmt.Sprintf("write model %s: %v", modelKey, err))
			}
		}
	}

	p.writeGatewayLog(ctx, msg, len(issues) == 0, strings.Join(issues, "; "))
	return nil
}

func (p *DefaultPropertyProcessor) gatewayDefinitions(ctx context.Context, modelKey string, cache map[string]map[string]tsl.Property) (map[string]tsl.Property, error) {
	if definitions, ok := cache[modelKey]; ok {
		return definitions, nil
	}
	if p.resolver == nil {
		return nil, errors.New("property definition resolver is not configured")
	}
	definitions, err := p.resolver.GetPropertiesByKey(ctx, modelKey)
	if err != nil {
		return nil, err
	}
	cache[modelKey] = definitions
	return definitions, nil
}

func (p *DefaultPropertyProcessor) writeGatewayLog(ctx context.Context, msg *DeviceMessage, success bool, errorMsg string) {
	if msg == nil {
		return
	}
	if p == nil || p.logs == nil {
		logging.V(2).InfoS("Gateway property report processed", "gatewayId", msg.GatewayID, "success", success, "err", errorMsg)
		return
	}
	log := timeseries.GatewayPropertyLog{
		GatewayType: msg.GatewayType,
		GatewayID:   msg.GatewayID,
		Time:        time.Now(),
		Success:     success,
		Payload:     msg.Payload,
		ErrorMsg:    errorMsg,
	}
	if err := p.logs.WriteGatewayPropertyLog(ctx, log); err != nil && !errors.Is(err, timeseries.ErrDisabled) {
		logging.ErrorS(err, "Failed to write gateway property log", "gatewayId", msg.GatewayID)
	}
}
