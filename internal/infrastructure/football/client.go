package football

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fifawcp/api/internal/infrastructure/config"
)

type apiEnvelope[T any] struct {
	Response []T `json:"response"`
}

type FootballClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func NewFootballClient(cfg config.FootballAPIConfig) *FootballClient {
	return &FootballClient{
		apiKey:  cfg.Key,
		baseURL: cfg.BaseURL,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *FootballClient) GetFixture(ctx context.Context, fixtureID int64) (*FixtureResponse, error) {
	url := fmt.Sprintf("%s/fixtures?id=%d", c.baseURL, fixtureID)
	fixtures, err := get[FixtureResponse](ctx, c, url)
	if err != nil {
		return nil, err
	}
	if len(fixtures) == 0 {
		return nil, fmt.Errorf("fixture %d not found", fixtureID)
	}

	return &fixtures[0], nil
}

func (c *FootballClient) GetFixturesByTeam(ctx context.Context, teamAPIID int64) ([]FixtureResponse, error) {
	url := fmt.Sprintf("%s/fixtures?league=1&season=2026&team=%d", c.baseURL, teamAPIID)
	return get[FixtureResponse](ctx, c, url)
}

func get[T any](ctx context.Context, c *FootballClient, url string) ([]T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("x-apisports-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}

	var envelope apiEnvelope[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode response from %s: %w", url, err)
	}

	return envelope.Response, nil
}
