package commands

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"patchmon-agent/internal/client"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

// serveCmd runs the agent as a long-lived service
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the agent as a service with async updates",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRoot(); err != nil {
			return err
		}
		return runService()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runService() error {
	if err := cfgManager.LoadCredentials(); err != nil {
		return err
	}

	httpClient := client.New(cfgManager, logger)
	ctx := context.Background()

	// obtain initial interval
	intervalMinutes := 60
	if resp, err := httpClient.GetUpdateInterval(ctx); err == nil && resp.UpdateInterval > 0 {
		intervalMinutes = resp.UpdateInterval
	}

	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()

	// initial report on boot
	if err := sendReport(); err != nil {
		logger.WithError(err).Warn("initial report failed")
	}

	// start websocket loop
	messages := make(chan wsMsg, 10)
	go wsLoop(messages)

	for {
		select {
		case <-ticker.C:
			if err := sendReport(); err != nil {
				logger.WithError(err).Warn("periodic report failed")
			}
		case m := <-messages:
			switch m.kind {
			case "settings_update":
				if m.interval > 0 {
					ticker.Stop()
					ticker = time.NewTicker(time.Duration(m.interval) * time.Minute)
					logger.WithField("new_interval", m.interval).Info("interval updated, no report sent")
				}
			case "report_now":
				if err := sendReport(); err != nil {
					logger.WithError(err).Warn("report_now failed")
				}
			case "update_agent":
				if err := updateAgent(); err != nil {
					logger.WithError(err).Warn("update_agent failed")
				}
			case "update_notification":
				logger.WithField("version", m.version).Info("Update notification received from server")
				if m.force {
					logger.Info("Force update requested, updating agent now")
					if err := updateAgent(); err != nil {
						logger.WithError(err).Warn("forced update failed")
					}
				} else {
					logger.Info("Update available, run 'patchmon-agent update-agent' to update")
				}
			}
		}
	}
}

type wsMsg struct {
	kind     string
	interval int
	version  string
	force    bool
}

func wsLoop(out chan<- wsMsg) {
	backoff := time.Second
	for {
		if err := connectOnce(out); err != nil {
			logger.WithError(err).Warn("ws disconnected; retrying")
		}
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func connectOnce(out chan<- wsMsg) error {
	server := cfgManager.GetConfig().PatchmonServer
	if server == "" {
		return nil
	}
	apiID := cfgManager.GetCredentials().APIID
	apiKey := cfgManager.GetCredentials().APIKey

	// Convert http(s) -> ws(s)
	wsURL := server
	if strings.HasPrefix(wsURL, "https://") {
		wsURL = "wss://" + strings.TrimPrefix(wsURL, "https://")
	} else if strings.HasPrefix(wsURL, "http://") {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	}
	if strings.HasSuffix(wsURL, "/") {
		wsURL = strings.TrimRight(wsURL, "/")
	}
	wsURL = wsURL + "/api/" + cfgManager.GetConfig().APIVersion + "/agents/ws"
	header := http.Header{}
	header.Set("X-API-ID", apiID)
	header.Set("X-API-KEY", apiKey)

	// Configure WebSocket dialer for insecure connections if needed
	dialer := websocket.DefaultDialer
	if cfgManager.GetConfig().SkipSSLVerify {
		dialer = &websocket.Dialer{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		logger.Warn("⚠️  SSL certificate verification is disabled for WebSocket")
	}

	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	// ping loop
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
		}
	}()

	// Set read deadlines and extend them on pong frames to avoid idle timeouts
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	})

	logger.WithField("url", wsURL).Info("WebSocket connected")
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var payload struct {
			Type           string `json:"type"`
			UpdateInterval int    `json:"update_interval"`
			Version        string `json:"version"`
			Force          bool   `json:"force"`
			Message        string `json:"message"`
		}
		if json.Unmarshal(data, &payload) == nil {
			switch payload.Type {
			case "settings_update":
				logger.WithField("interval", payload.UpdateInterval).Info("settings_update received")
				out <- wsMsg{kind: "settings_update", interval: payload.UpdateInterval}
			case "report_now":
				logger.Info("report_now received")
				out <- wsMsg{kind: "report_now"}
			case "update_agent":
				logger.Info("update_agent received")
				out <- wsMsg{kind: "update_agent"}
			case "update_notification":
				logger.WithFields(map[string]interface{}{
					"version": payload.Version,
					"force":   payload.Force,
					"message": payload.Message,
				}).Info("update_notification received")
				out <- wsMsg{
					kind:    "update_notification",
					version: payload.Version,
					force:   payload.Force,
				}
			}
		}
	}
}
