package opcua

import (
	"context"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

type uaClient struct {
	client *opcua.Client
}

func (c *uaClient) Available() bool {
	if c == nil || c.client == nil {
		return false
	}
	return c.client.State() != opcua.Closed && c.client.State() != opcua.Disconnected
}

func (c *uaClient) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close(context.Background())
}

func newUAClient(ctx context.Context, endpoint, username, password string) (*uaClient, error) {
	endpoints, err := opcua.GetEndpoints(ctx, endpoint)
	if err == nil {
		var endpointDescription *ua.EndpointDescription
		for _, item := range endpoints {
			if item.SecurityMode == ua.MessageSecurityModeNone {
				endpointDescription = item
				break
			}
		}
		opts := make([]opcua.Option, 0, 2)
		if username != "" && password != "" {
			opts = append(opts, opcua.AuthUsername(username, password))
			if endpointDescription != nil {
				opts = append(opts, opcua.SecurityFromEndpoint(endpointDescription, ua.UserTokenTypeUserName))
			} else {
				opts = append(opts, opcua.SecurityMode(ua.MessageSecurityModeNone))
			}
		} else if endpointDescription != nil {
			opts = append(opts, opcua.SecurityFromEndpoint(endpointDescription, ua.UserTokenTypeAnonymous))
		} else {
			opts = append(opts, opcua.SecurityMode(ua.MessageSecurityModeNone))
		}
		client, err := opcua.NewClient(endpoint, opts...)
		if err != nil {
			return nil, err
		}
		if err := client.Connect(ctx); err != nil {
			return nil, err
		}
		return &uaClient{client: client}, nil
	}

	opts := []opcua.Option{opcua.SecurityMode(ua.MessageSecurityModeNone)}
	if username != "" && password != "" {
		opts = append(opts, opcua.AuthUsername(username, password))
	}
	client, err := opcua.NewClient(endpoint, opts...)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	return &uaClient{client: client}, nil
}
