package opcua

import (
	"context"
	"errors"
	"fmt"

	"github.com/gopcua/opcua/ua"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/collection/connpool"
	"harns-platform/pkg/gateway/validation"
)

// DeliverAction writes collection variables through OPC UA Write.
func (c *OpcUaCollection) DeliverAction(ctx context.Context, actions []collection.ActionValue) error {
	runtime := c.runtime
	if runtime == nil || runtime.pool == nil {
		return validation.New([]validation.FieldIssue{{Field: "collection", MsgID: "validate.action_collection_not_ready"}})
	}
	if len(actions) == 0 {
		return nil
	}
	request, err := buildWriteRequest(actions)
	if err != nil {
		return validation.New([]validation.FieldIssue{{
			Field: "actions", MsgID: "validate.action_value_invalid", Args: []interface{}{"", err.Error()},
		}})
	}
	var response *ua.WriteResponse
	err = runtime.pool.DoWithRetry(ctx, connpool.DefaultWriteAttempts, func(client connpool.Client) (bool, error) {
		conn := client.(*uaClient)
		resp, writeErr := conn.client.Write(ctx, request)
		if writeErr != nil {
			return true, writeErr
		}
		response = resp
		return false, nil
	})
	if err != nil {
		return collection.DeviceActionError(actions[0], err)
	}
	if response == nil || len(response.Results) != len(actions) {
		return collection.DeviceActionError(actions[0], fmt.Errorf("opcUa write response length mismatch"))
	}
	var errs []error
	for i, status := range response.Results {
		if status != ua.StatusOK {
			errs = append(errs, fmt.Errorf("opcUa node %s write failed: %s", actions[i].Variable.Address, status))
			continue
		}
		c.GetCollectionMeta().SetVariableValue(actions[i].Variable.Name, collection.ApplyTransform(actions[i].Value, actions[i].Variable))
	}
	if len(errs) > 0 {
		return collection.DeviceActionError(actions[0], errors.Join(errs...))
	}
	return nil
}

func buildWriteRequest(actions []collection.ActionValue) (*ua.WriteRequest, error) {
	nodes := make([]*ua.WriteValue, 0, len(actions))
	for _, action := range actions {
		if action.Variable == nil {
			return nil, fmt.Errorf("action variable is nil")
		}
		nodeID, err := parseNodeID(action.Variable.Address)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", action.Variable.Name, err)
		}
		variant, err := ua.NewVariant(action.Value)
		if err != nil {
			return nil, fmt.Errorf("variable %s: %w", action.Variable.Name, err)
		}
		nodes = append(nodes, &ua.WriteValue{
			NodeID:      nodeID,
			AttributeID: ua.AttributeIDValue,
			Value: &ua.DataValue{
				EncodingMask: ua.DataValueValue,
				Value:        variant,
			},
		})
	}
	return &ua.WriteRequest{NodesToWrite: nodes}, nil
}

var _ collection.ActionHandler = (*OpcUaCollection)(nil)
