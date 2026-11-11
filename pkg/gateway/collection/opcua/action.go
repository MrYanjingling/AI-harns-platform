package opcua

import (
	"context"
	"errors"
	"fmt"

	"github.com/gopcua/opcua/ua"

	"harns-platform/pkg/gateway/collection"
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
	client, err := runtime.pool.Get(ctx)
	if err != nil {
		return err
	}
	conn := client.(*uaClient)
	response, err := conn.client.Write(ctx, request)
	runtime.pool.Put(client, err != nil)
	if err != nil {
		return err
	}
	if response == nil || len(response.Results) != len(actions) {
		return fmt.Errorf("opcUa write response length mismatch")
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
		return errors.Join(errs...)
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
