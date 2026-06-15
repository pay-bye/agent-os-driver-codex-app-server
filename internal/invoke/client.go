package invoke

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pay-bye/agent-os-driver-codex-app-server/internal/config"
	"mime"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func (c *Client) Metadata(ctx context.Context) (Metadata, error) {
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, c.route("/compatibility"), nil)
	if err != nil {
		return Metadata{}, err
	}
	httpResponse, err := c.http.Do(httpRequest)
	if err != nil {
		return Metadata{}, fmt.Errorf("%w: %v", ErrEndpointUnavailable, err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		return Metadata{}, ErrEndpointUnavailable
	}
	if !jsonMediaType(httpResponse.Header.Get("Content-Type")) {
		return Metadata{}, ErrMetadataMalformed
	}
	var metadata Metadata
	decoder := json.NewDecoder(httpResponse.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return Metadata{}, fmt.Errorf("%w: %v", ErrMetadataMalformed, err)
	}
	if err := metadata.Validate(); err != nil {
		return Metadata{}, err
	}
	return metadata, nil
}

func (c *Client) route(path string) string {
	return c.baseURL + path
}

func (c *Client) Claim(ctx context.Context, channel string, leaseID string, leaseSeconds int) (Claim, error) {
	var response claimResponse
	err := c.post(ctx, "/claim", claimRequest{
		Channel: channel,
		Lease:   leaseID,
		Seconds: leaseSeconds,
	}, &response)
	if err != nil {
		return Claim{}, err
	}
	return claimFromResponse(response)
}

func (c *Client) post(ctx context.Context, route string, request any, response any) error {
	content, err := json.Marshal(request)
	if err != nil {
		return err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.route(route), bytes.NewReader(content))
	if err != nil {
		return err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpResponse, err := c.http.Do(httpRequest)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		return fmt.Errorf("invocation_route_failed: %s", route)
	}
	if response == nil {
		return nil
	}
	return json.NewDecoder(httpResponse.Body).Decode(response)
}

func (c *Client) Extend(ctx context.Context, leaseID string, token string, expiresAt time.Time) error {
	return c.post(ctx, "/extend", extendRequest{
		Lease:     leaseID,
		Token:     token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil)
}

func (c *Client) Ack(ctx context.Context, leaseID string, token string, needs []config.Need) error {
	return c.post(ctx, "/ack", completionRequest{
		Lease: leaseID,
		Token: token,
		Needs: needs,
	}, nil)
}

func (c *Client) Nack(ctx context.Context, leaseID string, token string, failure Payload, needs []config.Need) error {
	return c.post(ctx, "/nack", failureRequest{
		Lease:   leaseID,
		Token:   token,
		Failure: failure,
		Needs:   needs,
	}, nil)
}

func New(baseURL string, client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{baseURL: stringsTrimRightSlash(baseURL), http: client}
}

func stringsTrimRightSlash(value string) string {
	item, err := url.Parse(value)
	if err != nil || item.Path == "" || item.Path == "/" {
		return trimSlash(value)
	}
	item.Path = trimSlash(item.Path)
	return item.String()
}

func trimSlash(value string) string {
	for len(value) > 0 && value[len(value)-1] == '/' {
		value = value[:len(value)-1]
	}
	return value
}

func jsonMediaType(header string) bool {
	mediaType, _, err := mime.ParseMediaType(header)
	return err == nil && mediaType == "application/json"
}
