package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	webrtcmanager "golang-webrtc-streaming/internal/webrtc"

	"github.com/gin-gonic/gin"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

type Server struct {
	port          int
	webrtcManager *webrtcmanager.Manager
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
	RTMP struct {
		Connected bool `json:"connected"`
	} `json:"rtmp"`
}

func NewServer(port int, webrtcManager *webrtcmanager.Manager) *Server {
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
	}

	// Static files
	s.router.Static("/static", "./web/static")
	s.router.LoadHTMLGlob("web/templates/*")
	s.router.GET("/", s.handleIndex)
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

func (s *Server) handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Go WebRTC Streaming",
	})
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
	// Get the latest frame from video stream
	// This is a simplified implementation - in production you'd want to cache frames

	peers := s.webrtcManager.GetAllPeers()
	if len(peers) == 0 {
		c.JSON(http.StatusServiceUnavailable, SnapshotResponse{
			Success: false,
			Error:   "No active streams available",
		})
		return
	}

	// For now, return a placeholder response
	// In a real implementation, you'd capture the current frame
	response := SnapshotResponse{
		Success: true,
		Data:    "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k=",
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
		RTMP: struct {
			Connected bool `json:"connected"`
		}{
			Connected: true, // RTMP server is always "connected" when running
		},
	}

	c.JSON(http.StatusOK, response)
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
