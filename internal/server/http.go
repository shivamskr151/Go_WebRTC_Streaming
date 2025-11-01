package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang-webrtc-streaming/internal/source"
	webrtcmanager "golang-webrtc-streaming/internal/webrtc"

	"github.com/gin-gonic/gin"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

type Server struct {
	port          int
	webrtcManager *webrtcmanager.Manager
	sourceManager *source.Manager
	router        *gin.Engine
	server        *http.Server
	isRunning     bool
	mu            sync.RWMutex
}

type OfferRequest struct {
	SDP webrtc.SessionDescription `json:"sdp"`
}

type OfferResponse struct {
	SDP string `json:"sdp"`
}

type SnapshotResponse struct {
	Success bool   `json:"success"`
	Data    string `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type StatusResponse struct {
	WebRTC struct {
		ConnectedPeers int `json:"connected_peers"`
		TotalPeers     int `json:"total_peers"`
	} `json:"webrtc"`
	Source struct {
		Type      string   `json:"type"`
		Running   bool     `json:"running"`
		Available []string `json:"available"`
	} `json:"source"`
	Streams struct {
		RTMP bool `json:"rtmp"`
		RTSP bool `json:"rtsp"`
	} `json:"streams"`
}

type SourceSwitchRequest struct {
	Type string `json:"type"`
}

func NewServer(port int, webrtcManager *webrtcmanager.Manager, sourceManager *source.Manager) *Server {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	// Enable CORS
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	server := &Server{
		port:          port,
		webrtcManager: webrtcManager,
		sourceManager: sourceManager,
		router:        router,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// API routes
	api := s.router.Group("/api")
	{
		api.POST("/offer", s.handleOffer)
		api.GET("/snapshot", s.handleSnapshot)
		api.GET("/status", s.handleStatus)
		api.GET("/peers", s.handlePeers)
		api.GET("/source", s.handleGetSource)
		api.POST("/source", s.handleSwitchSource)
		api.GET("/debug", s.handleDebug)
	}

	// Serve React static files
	s.router.Static("/assets", "./web/dist/assets")
	s.router.StaticFile("/", "./web/dist/index.html")
	s.router.StaticFile("/index.html", "./web/dist/index.html")

	// Fallback to index.html for client-side routing
	s.router.NoRoute(func(c *gin.Context) {
		// Only serve index.html for non-API routes
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] != "/api" {
			c.File("./web/dist/index.html")
		} else {
			c.Status(http.StatusNotFound)
		}
	})
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("HTTP server is already running")
	}

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("HTTP server error: %v", err)
		}
	}()

	s.isRunning = true
	logrus.Infof("HTTP server started on port %d", s.port)

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		logrus.Errorf("HTTP server shutdown error: %v", err)
	}

	s.isRunning = false
	logrus.Info("HTTP server stopped")
	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning || s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	s.isRunning = false
	logrus.Info("HTTP server stopped")
	return nil
}

func (s *Server) handleOffer(c *gin.Context) {
	var req OfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Parse the offer
	offer := req.SDP

	// Generate peer ID
	peerID := fmt.Sprintf("peer_%d", time.Now().UnixNano())

	// Ensure video source is running when first peer connects
	// Default to RTSP as it's more reliable for MediaMTX
	currentSource := s.sourceManager.GetCurrentSource()
	if currentSource == "" {
		// No source set, default to RTSP
		currentSource = "rtsp"
	}

	// Start source if not running
	if !s.sourceManager.IsSourceRunning() {
		logrus.Infof("Starting source %s for new peer connection", currentSource)
		if err := s.sourceManager.StartSource(c.Request.Context(), currentSource); err != nil {
			logrus.Errorf("Failed to start source %s: %v", currentSource, err)
			// Try RTSP as fallback if current source failed
			if currentSource != "rtsp" {
				logrus.Infof("Attempting RTSP as fallback")
				if err := s.sourceManager.StartSource(c.Request.Context(), "rtsp"); err != nil {
					logrus.Errorf("Failed to start RTSP source: %v", err)
					c.JSON(http.StatusServiceUnavailable, gin.H{
						"error": fmt.Sprintf("Video source unavailable. RTSP error: %v", err),
					})
					return
				}
				currentSource = "rtsp"
			} else {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": fmt.Sprintf("Failed to start video source: %v", err),
				})
				return
			}
		}
		// Give source a moment to start streaming
		time.Sleep(100 * time.Millisecond)
	}

	// Create peer
	_, err := s.webrtcManager.CreatePeer(peerID)
	if err != nil {
		logrus.Errorf("Failed to create peer: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create peer"})
		return
	}

	// Handle the offer
	answer, err := s.webrtcManager.HandleOffer(peerID, offer)
	if err != nil {
		logrus.Errorf("Failed to handle offer: %v", err)
		s.webrtcManager.RemovePeer(peerID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to handle offer"})
		return
	}

	// Return the answer directly without double JSON encoding
	response := OfferResponse{
		SDP: answer.SDP,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) handleSnapshot(c *gin.Context) {
	// Check if there are active streams
	peers := s.webrtcManager.GetAllPeers()
	if len(peers) == 0 {
		c.JSON(http.StatusServiceUnavailable, SnapshotResponse{
			Success: false,
			Error:   "No active streams available",
		})
		return
	}

	// Capture snapshot from the latest video frame
	snapshotData, err := s.webrtcManager.CaptureSnapshot()
	if err != nil {
		logrus.Errorf("Failed to capture snapshot: %v", err)
		c.JSON(http.StatusInternalServerError, SnapshotResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to capture snapshot: %v", err),
		})
		return
	}

	response := SnapshotResponse{
		Success: true,
		Data:    snapshotData,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) handleStatus(c *gin.Context) {
	peers := s.webrtcManager.GetAllPeers()
	connectedPeers := s.webrtcManager.GetConnectedPeersCount()

	response := StatusResponse{
		WebRTC: struct {
			ConnectedPeers int `json:"connected_peers"`
			TotalPeers     int `json:"total_peers"`
		}{
			ConnectedPeers: connectedPeers,
			TotalPeers:     len(peers),
		},
		Source: struct {
			Type      string   `json:"type"`
			Running   bool     `json:"running"`
			Available []string `json:"available"`
		}{
			Type:      s.sourceManager.GetCurrentSource(),
			Running:   s.sourceManager.IsSourceRunning(),
			Available: s.sourceManager.GetAvailableSources(),
		},
		Streams: struct {
			RTMP bool `json:"rtmp"`
			RTSP bool `json:"rtsp"`
		}{
			RTMP: s.sourceManager != nil && len(filter(s.sourceManager.GetAvailableSources(), "rtmp")) > 0,
			RTSP: s.sourceManager != nil && len(filter(s.sourceManager.GetAvailableSources(), "rtsp")) > 0,
		},
	}

	c.JSON(http.StatusOK, response)
}

// helper
func filter(arr []string, v string) []string {
	out := make([]string, 0, len(arr))
	for _, s := range arr {
		if s == v {
			out = append(out, s)
		}
	}
	return out
}

func (s *Server) handlePeers(c *gin.Context) {
	peers := s.webrtcManager.GetAllPeers()

	peerList := make([]gin.H, 0, len(peers))
	for id, peer := range peers {
		peerList = append(peerList, gin.H{
			"id":               id,
			"connected":        peer.IsConnected,
			"connection_state": peer.Connection.ConnectionState().String(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"peers": peerList,
		"count": len(peers),
	})
}

func (s *Server) handleGetSource(c *gin.Context) {
	response := gin.H{
		"type":      s.sourceManager.GetCurrentSource(),
		"running":   s.sourceManager.IsSourceRunning(),
		"available": s.sourceManager.GetAvailableSources(),
	}
	c.JSON(http.StatusOK, response)
}

func (s *Server) handleSwitchSource(c *gin.Context) {
	var req SourceSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Switch source (case-insensitive, with lazy init in manager)
	if err := s.sourceManager.StartSource(c.Request.Context(), req.Type); err != nil {
		logrus.Errorf("Failed to switch to %s source: %v", req.Type, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     fmt.Sprintf("Failed to switch to %s: %v", req.Type, err),
			"available": s.sourceManager.GetAvailableSources(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Switched to %s source", req.Type),
		"type":    req.Type,
	})
}

func (s *Server) handleDebug(c *gin.Context) {
	peers := s.webrtcManager.GetAllPeers()
	connectedPeers := s.webrtcManager.GetConnectedPeersCount()

	peerDetails := make([]gin.H, 0, len(peers))
	for id, peer := range peers {
		connState := "unknown"
		iceState := "unknown"
		hasVideoTrack := peer.VideoTrack != nil
		isConnected := peer.IsConnected
		if peer.Connection != nil {
			connState = peer.Connection.ConnectionState().String()
			iceState = peer.Connection.ICEConnectionState().String()
		}
		peerDetails = append(peerDetails, gin.H{
			"id":               id,
			"has_video_track":  hasVideoTrack,
			"connection_state": connState,
			"ice_state":        iceState,
			"is_connected":     isConnected,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"webrtc": gin.H{
			"total_peers":     len(peers),
			"connected_peers": connectedPeers,
			"peers":           peerDetails,
		},
		"source": gin.H{
			"type":      s.sourceManager.GetCurrentSource(),
			"running":   s.sourceManager.IsSourceRunning(),
			"available": s.sourceManager.GetAvailableSources(),
		},
	})
}
