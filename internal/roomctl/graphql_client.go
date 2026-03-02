package roomctl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type PublishMutationInput struct {
	ClientRequestID  string `json:"clientRequestId"`
	RoomSlug         string `json:"roomSlug"`
	Version          int    `json:"version"`
	Changelog        string `json:"changelog"`
	BundleHashSha256 string `json:"bundleHashSha256"`
	Activate         bool   `json:"activate"`
}

type PublishedRoomVersion struct {
	ID            string `json:"id"`
	VersionNumber int    `json:"versionNumber"`
	Status        string `json:"status"`
	Changelog     string `json:"changelog"`
}

type GraphQLClient struct {
	url         string
	adminAPIKey string
	httpClient  *http.Client
}

func NewGraphQLClient(url, adminAPIKey string) *GraphQLClient {
	return &GraphQLClient{
		url:         strings.TrimSpace(url),
		adminAPIKey: strings.TrimSpace(adminAPIKey),
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *GraphQLClient) PublishRoomVersion(ctx context.Context, input PublishMutationInput) (*PublishedRoomVersion, error) {
	const query = `mutation PublishRoomVersion($input: PublishRoomVersionInput!) { publishRoomVersion(input: $input) { roomVersion { id versionNumber status changelog } } }`

	body, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": map[string]any{"input": input},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.adminAPIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return nil, fmt.Errorf("graphql http %d: %s", resp.StatusCode, msg)
	}

	var parsed struct {
		Data *struct {
			PublishRoomVersion *struct {
				RoomVersion *PublishedRoomVersion `json:"roomVersion"`
			} `json:"publishRoomVersion"`
		} `json:"data"`
		Errors []struct {
			Message    string         `json:"message"`
			Extensions map[string]any `json:"extensions"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Errors) > 0 {
		code, _ := parsed.Errors[0].Extensions["code"].(string)
		if code != "" {
			return nil, fmt.Errorf("graphql error (%s): %s", code, parsed.Errors[0].Message)
		}
		return nil, fmt.Errorf("graphql error: %s", parsed.Errors[0].Message)
	}
	if parsed.Data == nil || parsed.Data.PublishRoomVersion == nil || parsed.Data.PublishRoomVersion.RoomVersion == nil {
		return nil, fmt.Errorf("graphql response missing publishRoomVersion.roomVersion")
	}

	return parsed.Data.PublishRoomVersion.RoomVersion, nil
}
