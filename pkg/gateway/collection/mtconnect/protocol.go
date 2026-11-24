package mtconnect

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"harns-platform/pkg/gateway/collection"
)

type mtcClient struct {
	baseURL  string
	client   *http.Client
	timeout  time.Duration
	nsPrefix string
	nsURI    string
	closed   bool
}

func openMTCClient(host string, port int, nsPrefix, nsURI string, timeout time.Duration) (*mtcClient, error) {
	if strings.TrimSpace(host) == "" || port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid MTConnect endpoint")
	}
	transport := &http.Transport{Proxy: nil, MaxIdleConns: 2, MaxIdleConnsPerHost: 1, IdleConnTimeout: 30 * time.Second, ResponseHeaderTimeout: timeout}
	return &mtcClient{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		client:  &http.Client{Transport: transport, Timeout: timeout},
		timeout: timeout, nsPrefix: nsPrefix, nsURI: nsURI,
	}, nil
}

func (c *mtcClient) Available() bool { return c != nil && !c.closed }
func (c *mtcClient) Close() error {
	if c == nil {
		return nil
	}
	c.closed = true
	if transport, ok := c.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

func (c *mtcClient) Read(ctx context.Context, address string, dataType string) (any, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("MTConnect client is not initialized")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/current", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/xml")
	response, err := c.client.Do(request)
	if err != nil {
		return nil, collection.ConnectionLost(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MTConnect HTTP status %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, collection.ConnectionLost(err)
	}
	root, err := parseXML(data)
	if err != nil {
		return nil, fmt.Errorf("invalid MTConnect XML: %w", err)
	}
	text, err := findXPath(root, address)
	if err != nil {
		return nil, err
	}
	return convertValue(text, dataType)
}
