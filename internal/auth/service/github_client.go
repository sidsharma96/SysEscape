package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	githubOAuthTokenURL = "https://github.com/login/oauth/access_token"
	githubUserAPIURL    = "https://api.github.com/user"
)

// GitHubClient defines GitHub OAuth and user profile operations.
type GitHubClient interface {
	ExchangeCode(ctx context.Context, code string) (string, error)
	GetUser(ctx context.Context, accessToken string) (GitHubUser, error)
}

// GitHubUser is the minimal profile fetched from GitHub.
type GitHubUser struct {
	ID    int64
	Login string
	Name  string
}

// RealGitHubClient calls the real GitHub OAuth and API endpoints.
type RealGitHubClient struct {
	httpClient *http.Client
	config     OAuthConfig
}

// NewRealGitHubClient returns a RealGitHubClient.
func NewRealGitHubClient(httpClient *http.Client, config OAuthConfig) *RealGitHubClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &RealGitHubClient{httpClient: httpClient, config: config}
}

func (c *RealGitHubClient) ExchangeCode(ctx context.Context, code string) (string, error) {
	if code == "" {
		return "", errors.New("oauth code is required")
	}

	form := url.Values{}
	form.Set("client_id", c.config.ClientID)
	form.Set("client_secret", c.config.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", c.config.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute token exchange request: %w", err)
	}
	defer resp.Body.Close()

	var payload struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode token exchange response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("github token exchange failed: status %d", resp.StatusCode)
	}
	if payload.Error != "" {
		if payload.ErrorDescription != "" {
			return "", fmt.Errorf("github token exchange error: %s: %s", payload.Error, payload.ErrorDescription)
		}
		return "", fmt.Errorf("github token exchange error: %s", payload.Error)
	}
	if payload.AccessToken == "" {
		return "", errors.New("github token exchange returned empty access token")
	}

	return payload.AccessToken, nil
}

func (c *RealGitHubClient) GetUser(ctx context.Context, accessToken string) (GitHubUser, error) {
	if accessToken == "" {
		return GitHubUser{}, errors.New("access token is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUserAPIURL, nil)
	if err != nil {
		return GitHubUser{}, fmt.Errorf("build user request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return GitHubUser{}, fmt.Errorf("execute user request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return GitHubUser{}, fmt.Errorf("github user request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return GitHubUser{}, fmt.Errorf("decode github user response: %w", err)
	}
	if payload.ID == 0 || payload.Login == "" {
		return GitHubUser{}, errors.New("github user response missing required fields")
	}

	return GitHubUser{
		ID:    payload.ID,
		Login: payload.Login,
		Name:  payload.Name,
	}, nil
}

// MockGitHubClient is a test double for GitHubClient.
type MockGitHubClient struct {
	ExchangeCodeFunc func(ctx context.Context, code string) (string, error)
	GetUserFunc      func(ctx context.Context, accessToken string) (GitHubUser, error)
}

func (m *MockGitHubClient) ExchangeCode(ctx context.Context, code string) (string, error) {
	if m.ExchangeCodeFunc == nil {
		return "", nil
	}
	return m.ExchangeCodeFunc(ctx, code)
}

func (m *MockGitHubClient) GetUser(ctx context.Context, accessToken string) (GitHubUser, error) {
	if m.GetUserFunc == nil {
		return GitHubUser{}, nil
	}
	return m.GetUserFunc(ctx, accessToken)
}
