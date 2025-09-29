package client

import (
	"context"
	"fmt"
	"time"

	"patchmon-agent/internal/config"
	"patchmon-agent/pkg/models"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

// Client handles HTTP communications with the PatchMon server
type Client struct {
	client      *resty.Client
	config      *models.Config
	credentials *models.Credentials
	logger      *logrus.Logger
}

// New creates a new HTTP client
func New(configMgr *config.Manager, logger *logrus.Logger) *Client {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(2 * time.Second)

	return &Client{
		client:      client,
		config:      configMgr.GetConfig(),
		credentials: configMgr.GetCredentials(),
		logger:      logger,
	}
}

// Ping sends a ping request to the server
func (c *Client) Ping(ctx context.Context) (*models.PingResponse, error) {
	url := fmt.Sprintf("%s/api/%s/hosts/ping", c.config.PatchmonServer, c.config.APIVersion)

	c.logger.Debug("Sending ping request to server")

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-API-ID", c.credentials.APIID).
		SetHeader("X-API-KEY", c.credentials.APIKey).
		SetResult(&models.PingResponse{}).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("ping request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("ping request failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	result, ok := resp.Result().(*models.PingResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return result, nil
}

// SendUpdate sends package update information to the server
func (c *Client) SendUpdate(ctx context.Context, payload *models.UpdatePayload) (*models.UpdateResponse, error) {
	url := fmt.Sprintf("%s/api/%s/hosts/update", c.config.PatchmonServer, c.config.APIVersion)

	c.logger.Debug("Sending update to server")

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-API-ID", c.credentials.APIID).
		SetHeader("X-API-KEY", c.credentials.APIKey).
		SetBody(payload).
		SetResult(&models.UpdateResponse{}).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("update request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("update request failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	result, ok := resp.Result().(*models.UpdateResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	if !result.Success {
		return nil, fmt.Errorf("update failed: server returned success=false")
	}

	return result, nil
}

// CheckVersion checks for agent version updates
func (c *Client) CheckVersion(ctx context.Context) (*models.VersionResponse, error) {
	url := fmt.Sprintf("%s/api/%s/hosts/agent/version", c.config.PatchmonServer, c.config.APIVersion)

	c.logger.Debug("Checking for version updates")

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&models.VersionResponse{}).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("version check failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("version check failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	result, ok := resp.Result().(*models.VersionResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return result, nil
}

// DownloadUpdate downloads the latest agent version
func (c *Client) DownloadUpdate(ctx context.Context, downloadURL string) ([]byte, error) {
	// If download URL is relative, make it absolute
	if len(downloadURL) > 0 && downloadURL[0] == '/' {
		downloadURL = c.config.PatchmonServer + downloadURL
	}

	// If no specific download URL, use default
	if downloadURL == "" {
		downloadURL = fmt.Sprintf("%s/api/%s/hosts/agent/download", c.config.PatchmonServer, c.config.APIVersion)
	}

	c.logger.Debugf("Downloading agent update from: %s", downloadURL)

	resp, err := c.client.R().SetContext(ctx).Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("download failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	return resp.Body(), nil
}

// GetUpdateInterval gets the current update interval from server
func (c *Client) GetUpdateInterval(ctx context.Context) (*models.UpdateIntervalResponse, error) {
	url := fmt.Sprintf("%s/api/%s/settings/update-interval", c.config.PatchmonServer, c.config.APIVersion)

	c.logger.Debug("Getting update interval from server")

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&models.UpdateIntervalResponse{}).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("update interval request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("update interval request failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	result, ok := resp.Result().(*models.UpdateIntervalResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return result, nil
}
