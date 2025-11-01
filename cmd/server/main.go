package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang-webrtc-streaming/internal/config"
	"golang-webrtc-streaming/internal/rtmp"
	"golang-webrtc-streaming/internal/server"
	"golang-webrtc-streaming/internal/source"
	"golang-webrtc-streaming/internal/webrtc"

	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize logger
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Load .env early (project root)
	config.LoadDotEnv(".env")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize WebRTC manager
	webrtcManager := webrtc.NewManager()

	// Initialize source manager with MediaMTX configuration
	// MediaMTX ingests from cameras, we pull optimized streams from MediaMTX
	sourceManager := source.NewManager(webrtcManager)
	sourceManager.InitializeSources(
		cfg.MediaMTX.Host,
		cfg.MediaMTX.RTSPPort,
		cfg.MediaMTX.RTMPPort,
	)

	// Initialize RTMP server
	rtmpServer := rtmp.NewServer(cfg.RTMP.Port, webrtcManager)

	// Initialize HTTP server with source manager
	httpServer := server.NewServer(cfg.HTTP.Port, webrtcManager, sourceManager)

	// Start all configured sources, select active type if provided
	sourceManager.StartAll(ctx)

	// Set default active source - prefer RTSP for MediaMTX
	defaultSource := "rtsp"
	if cfg.Source.Type != "" {
		defaultSource = cfg.Source.Type
	} else if cfg.RTMP.URL != "" && cfg.RTSP.URL == "" {
		defaultSource = "rtmp"
	}

	if err := sourceManager.SetActiveSource(defaultSource); err != nil {
		logrus.Warnf("Failed to set active source: %v", err)
	} else {
		logrus.Infof("✅ Active source set to: %s", defaultSource)
		// Start the active source
		if err := sourceManager.StartSource(ctx, defaultSource); err != nil {
			logrus.Warnf("Failed to start default source %s: %v", defaultSource, err)
		}
	}

	// Start RTMP server
	go func() {
		if err := rtmpServer.Start(ctx); err != nil {
			logrus.Errorf("RTMP server error: %v", err)
		}
	}()

	// Start HTTP server
	go func() {
		if err := httpServer.Start(ctx); err != nil {
			logrus.Errorf("HTTP server error: %v", err)
		}
	}()

	// Print startup information
	printStartupInfo(cfg)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logrus.Info("Shutting down gracefully...")
	cancel()

	// Give services time to shutdown
	time.Sleep(2 * time.Second)
	logrus.Info("Shutdown complete")
}

func printStartupInfo(cfg *config.Config) {
	fmt.Println("🚀 Go WebRTC Streaming Server Started")
	fmt.Println("=====================================")
	fmt.Printf("📡 HTTP Server: http://localhost:%d\n", cfg.HTTP.Port)
	fmt.Printf("📺 RTMP Server: rtmp://localhost:%d/live\n", cfg.RTMP.Port)
	fmt.Printf("🎬 MediaMTX Host: %s:%d (RTSP), %s:%d (RTMP)\n",
		cfg.MediaMTX.Host, cfg.MediaMTX.RTSPPort,
		cfg.MediaMTX.Host, cfg.MediaMTX.RTMPPort)

	// Show MediaMTX URLs (where Go server pulls from)
	fmt.Printf("📥 Pulling RTMP from MediaMTX: rtmp://%s:%d/live\n",
		cfg.MediaMTX.Host, cfg.MediaMTX.RTMPPort)
	fmt.Printf("📥 Pulling RTSP from MediaMTX: rtsp://%s:%d/live\n",
		cfg.MediaMTX.Host, cfg.MediaMTX.RTSPPort)

	// Show original camera sources (where MediaMTX ingests from)
	if cfg.RTMP.URL != "" {
		fmt.Printf("📹 Camera RTMP (→ MediaMTX): %s\n", cfg.RTMP.URL)
	}
	if cfg.RTSP.URL != "" {
		fmt.Printf("📹 Camera RTSP (→ MediaMTX): %s\n", cfg.RTSP.URL)
	}

	fmt.Println("🌐 Web Client: http://localhost:8080")
	fmt.Println("📸 Snapshot API: http://localhost:8080/api/snapshot")
	fmt.Println("🔄 Switch Source API: http://localhost:8080/api/source")
	fmt.Println("=====================================")
}
