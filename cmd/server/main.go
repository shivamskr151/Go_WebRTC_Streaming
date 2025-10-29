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
	"golang-webrtc-streaming/internal/webrtc"

	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize logger
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

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

	// Initialize RTMP client
	rtmpClient := rtmp.NewClient(cfg.RTMP.URL, webrtcManager)

	// Initialize RTMP server
	rtmpServer := rtmp.NewServer(cfg.RTMP.Port, webrtcManager)

	// Initialize HTTP server
	httpServer := server.NewServer(cfg.HTTP.Port, webrtcManager)

	// Start RTMP client
	go func() {
		if err := rtmpClient.Start(ctx); err != nil {
			logrus.Errorf("RTMP client error: %v", err)
		}
	}()

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
	fmt.Printf("📹 RTMP Stream: %s\n", cfg.RTMP.URL)
	fmt.Println("🌐 Web Client: http://localhost:8080")
	fmt.Println("📸 Snapshot API: http://localhost:8080/api/snapshot")
	fmt.Println("=====================================")
}
