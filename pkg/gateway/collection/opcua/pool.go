package opcua

import (
	"context"
	"fmt"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

type clientPool struct {
	endpoint string
	username string
	password string
	size     int

	idle chan *opcua.Client
}

func newClientPool(endpoint, username, password string, size int) (*clientPool, error) {
	if size < 1 {
		size = 1
	}
	return &clientPool{
		endpoint: endpoint,
		username: username,
		password: password,
		size:     size,
		idle:     make(chan *opcua.Client, size),
	}, nil
}

func (p *clientPool) connect(ctx context.Context) error {
	clients := make([]*opcua.Client, 0, p.size)
	for i := 0; i < p.size; i++ {
		client, err := p.newClient(ctx)
		if err != nil {
			for _, item := range clients {
				item.Close(context.Background())
			}
			return err
		}
		clients = append(clients, client)
	}
	for _, client := range clients {
		p.idle <- client
	}
	return nil
}

func (p *clientPool) acquire(ctx context.Context) (*opcua.Client, error) {
	select {
	case client := <-p.idle:
		if client == nil {
			return nil, fmt.Errorf("opcUa client pool closed")
		}
		return client, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *clientPool) release(client *opcua.Client, failed bool) {
	if client == nil {
		return
	}
	if failed {
		client.Close(context.Background())
		replacement, err := p.newClient(context.Background())
		if err == nil {
			p.idle <- replacement
		}
		return
	}
	p.idle <- client
}

func (p *clientPool) close(ctx context.Context) {
	for {
		select {
		case client := <-p.idle:
			if client != nil {
				client.Close(ctx)
			}
		default:
			return
		}
	}
}

func (p *clientPool) newClient(ctx context.Context) (*opcua.Client, error) {
	endpoints, err := opcua.GetEndpoints(ctx, p.endpoint)
	if err == nil {
		var endpoint *ua.EndpointDescription
		for _, item := range endpoints {
			if item.SecurityMode == ua.MessageSecurityModeNone {
				endpoint = item
				break
			}
		}
		opts := make([]opcua.Option, 0, 2)
		if p.username != "" && p.password != "" {
			opts = append(opts, opcua.AuthUsername(p.username, p.password))
			if endpoint != nil {
				opts = append(opts, opcua.SecurityFromEndpoint(endpoint, ua.UserTokenTypeUserName))
			} else {
				opts = append(opts, opcua.SecurityMode(ua.MessageSecurityModeNone))
			}
		} else if endpoint != nil {
			opts = append(opts, opcua.SecurityFromEndpoint(endpoint, ua.UserTokenTypeAnonymous))
		} else {
			opts = append(opts, opcua.SecurityMode(ua.MessageSecurityModeNone))
		}
		client, err := opcua.NewClient(p.endpoint, opts...)
		if err != nil {
			return nil, err
		}
		if err := client.Connect(ctx); err != nil {
			return nil, err
		}
		return client, nil
	}

	opts := []opcua.Option{opcua.SecurityMode(ua.MessageSecurityModeNone)}
	if p.username != "" && p.password != "" {
		opts = append(opts, opcua.AuthUsername(p.username, p.password))
	}
	client, err := opcua.NewClient(p.endpoint, opts...)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	return client, nil
}
