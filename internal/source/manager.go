package source

import (
	"context"
	"fmt"
	"sync"

	"golang-webrtc-streaming/internal/rtmp"
	"golang-webrtc-streaming/internal/rtsp"
	"golang-webrtc-streaming/internal/webrtc"

	"github.com/sirupsen/logrus"
)

type Manager struct {
	webrtcManager *webrtc.Manager
	rtmpClient    *rtmp.RTMPClient
	rtspClient    *rtsp.Client
	currentSource string
	rtmpURL       string
	rtspURL       string
	mu            sync.RWMutex
}

func NewManager(webrtcManager *webrtc.Manager) *Manager {
	return &Manager{
		webrtcManager: webrtcManager,
		currentSource: "",
	}
}

func (m *Manager) InitializeSources(rtmpURL, rtspURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rtmpURL = rtmpURL
	m.rtspURL = rtspURL

	if rtmpURL != "" {
		m.rtmpClient = rtmp.NewClient(rtmpURL, m.webrtcManager, func() bool {
			m.mu.RLock()
			defer m.mu.RUnlock()
			return m.currentSource == "rtmp"
		})
		logrus.Infof("Initialized RTMP client with URL: %s", rtmpURL)
	}

	if rtspURL != "" {
		m.rtspClient = rtsp.NewClient(rtspURL, m.webrtcManager, func() bool {
			m.mu.RLock()
			defer m.mu.RUnlock()
			return m.currentSource == "rtsp"
		})
		logrus.Infof("Initialized RTSP client with URL: %s", rtspURL)
	}
}

func (m *Manager) StartSource(ctx context.Context, sourceType string) error {
	m.mu.Lock()
	// Do not stop others; both run concurrently. Just switch active selector.

	switch normalize(sourceType) {
	case "rtmp":
		if m.rtmpClient == nil {
			if m.rtmpURL == "" {
				return fmt.Errorf("RTMP source not configured")
			}
			m.rtmpClient = rtmp.NewClient(m.rtmpURL, m.webrtcManager, func() bool {
				m.mu.RLock()
				defer m.mu.RUnlock()
				return m.currentSource == "rtmp"
			})
		}
		// Start if not running
		if !m.rtmpClient.IsRunning() {
			if err := m.rtmpClient.Start(ctx); err != nil {
				m.mu.Unlock()
				return fmt.Errorf("failed to start RTMP client: %w", err)
			}
		}
		m.currentSource = "rtmp"
		logrus.Info("✅ Started RTMP source")

	case "rtsp":
		if m.rtspClient == nil {
			if m.rtspURL == "" {
				return fmt.Errorf("RTSP source not configured")
			}
			m.rtspClient = rtsp.NewClient(m.rtspURL, m.webrtcManager, func() bool {
				m.mu.RLock()
				defer m.mu.RUnlock()
				return m.currentSource == "rtsp"
			})
		}
		if !m.rtspClient.IsRunning() {
			if err := m.rtspClient.Start(ctx); err != nil {
				m.mu.Unlock()
				return fmt.Errorf("failed to start RTSP client: %w", err)
			}
		}
		m.currentSource = "rtsp"
		logrus.Info("✅ Started RTSP source")

	default:
		m.mu.Unlock()
		return fmt.Errorf("unknown source type: %s", sourceType)
	}

	m.mu.Unlock()
	return nil
}

func (m *Manager) StopCurrentSource() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCurrentSource()
}

func (m *Manager) stopCurrentSource() {
	if m.currentSource == "" {
		return
	}

	switch m.currentSource {
	case "rtmp":
		if m.rtmpClient != nil {
			m.rtmpClient.Stop()
			logrus.Info("🛑 Stopped RTMP source")
		}
	case "rtsp":
		if m.rtspClient != nil {
			m.rtspClient.Stop()
			logrus.Info("🛑 Stopped RTSP source")
		}
	}
	m.currentSource = ""
}

func (m *Manager) GetCurrentSource() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentSource
}

func (m *Manager) GetAvailableSources() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sources []string
	if m.rtmpClient != nil || m.rtmpURL != "" {
		sources = append(sources, "rtmp")
	}
	if m.rtspClient != nil || m.rtspURL != "" {
		sources = append(sources, "rtsp")
	}
	return sources
}

func (m *Manager) IsSourceRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentSource == "" {
		return false
	}

	switch m.currentSource {
	case "rtmp":
		return m.rtmpClient != nil && m.rtmpClient.IsRunning()
	case "rtsp":
		return m.rtspClient != nil && m.rtspClient.IsRunning()
	}
	return false
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.rtmpClient != nil {
		m.rtmpClient.Stop()
	}
	if m.rtspClient != nil {
		m.rtspClient.Stop()
	}
	m.currentSource = ""
}

// StartAll starts both sources if configured. Active output is controlled by currentSource.
func (m *Manager) StartAll(ctx context.Context) {
	m.mu.Lock()
	rtsp := m.rtspClient
	rtmpc := m.rtmpClient
	m.mu.Unlock()

	if rtmpc != nil && !rtmpc.IsRunning() {
		go func() {
			if err := rtmpc.Start(ctx); err != nil {
				logrus.Errorf("RTMP client start error: %v", err)
			}
		}()
	}
	if rtsp != nil && !rtsp.IsRunning() {
		go func() {
			if err := rtsp.Start(ctx); err != nil {
				logrus.Errorf("RTSP client start error: %v", err)
			}
		}()
	}
}

// SetActiveSource switches the active output without starting/stopping clients.
func (m *Manager) SetActiveSource(sourceType string) error {
	st := normalize(sourceType)
	if st != "rtsp" && st != "rtmp" {
		return fmt.Errorf("unknown source type: %s", sourceType)
	}
	m.mu.Lock()
	m.currentSource = st
	m.mu.Unlock()
	return nil
}

func normalize(s string) string {
	switch s {
	case "RTMP", "rtmp", "Rtmp":
		return "rtmp"
	case "RTSP", "rtsp", "Rtsp":
		return "rtsp"
	default:
		return s
	}
}
