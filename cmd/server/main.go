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

	// Initialize source manager
	sourceManager := source.NewManager(webrtcManager)
	sourceManager.InitializeSources(cfg.RTMP.URL, cfg.RTSP.URL)

	// Initialize RTMP server
	rtmpServer := rtmp.NewServer(cfg.RTMP.Port, webrtcManager)

	// Initialize HTTP server with source manager
	httpServer := server.NewServer(cfg.HTTP.Port, webrtcManager, sourceManager)

	// Start all configured sources, select active type if provided
	sourceManager.StartAll(ctx)
	if cfg.Source.Type != "" {
		if err := sourceManager.SetActiveSource(cfg.Source.Type); err != nil {
			logrus.Warnf("Failed to set active source from config: %v", err)
		}
	} else if cfg.RTSP.URL != "" {
		_ = sourceManager.SetActiveSource("rtsp")
	} else if cfg.RTMP.URL != "" {
		_ = sourceManager.SetActiveSource("rtmp")
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

	// Show available sources
	if cfg.RTMP.URL != "" {
		fmt.Printf("📹 RTMP Source: %s\n", cfg.RTMP.URL)
	}
	if cfg.RTSP.URL != "" {
		fmt.Printf("📹 RTSP Source: %s\n", cfg.RTSP.URL)
	}
	if cfg.Source.URL != "" {
		fmt.Printf("🎯 Active Source: %s (%s)\n", cfg.Source.Type, cfg.Source.URL)
	}

	fmt.Println("🌐 Web Client: http://localhost:8080")
	fmt.Println("📸 Snapshot API: http://localhost:8080/api/snapshot")
	fmt.Println("🔄 Switch Source API: http://localhost:8080/api/source")
	fmt.Println("=====================================")
}
