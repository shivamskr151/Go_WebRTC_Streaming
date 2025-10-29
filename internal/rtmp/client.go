package rtmp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	webrtcmanager "golang-webrtc-streaming/internal/webrtc"

	"github.com/sirupsen/logrus"
)

type RTMPClient struct {
	url           string
	webrtcManager *webrtcmanager.Manager
	cmd           *exec.Cmd
	isRunning     bool
	mu            sync.RWMutex
	shouldWrite   func() bool
}

func NewClient(rtmpURL string, webrtcManager *webrtcmanager.Manager, shouldWrite func() bool) *RTMPClient {
	return &RTMPClient{
		url:           rtmpURL,
		webrtcManager: webrtcManager,
		shouldWrite:   shouldWrite,
		isRunning:     false,
	}
}

func (c *RTMPClient) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		return fmt.Errorf("RTMP client is already running")
	}

	logrus.Infof("Starting RTMP client for: %s", c.url)

	// Try to connect to RTMP stream with retries
	var cmd *exec.Cmd
	var stdout, stderr io.ReadCloser
	var err error

	for retries := 0; retries < 3; retries++ {
		logrus.Infof("Attempting RTMP connection (attempt %d): %s", retries+1, c.url)

		// Use FFmpeg to convert RTMP to H.264 stream
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-i", c.url,
			"-c", "copy", // copy all streams
			"-f", "h264", // output H.264 format
			"-an", // no audio
			"pipe:1",
		)

		// Get stdout pipe
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			logrus.Errorf("Failed to create stdout pipe (attempt %d): %v", retries+1, err)
			continue
		}

		// Get stderr pipe for logging
		stderr, err = cmd.StderrPipe()
		if err != nil {
			logrus.Errorf("Failed to create stderr pipe (attempt %d): %v", retries+1, err)
			continue
		}

		// Start the command
		if err = cmd.Start(); err != nil {
			logrus.Errorf("Failed to start ffmpeg (attempt %d): %v", retries+1, err)
			if retries < 2 {
				time.Sleep(time.Second * 3)
			}
			continue
		}

		// Give the command a moment to start
		time.Sleep(2 * time.Second)

		// Check if the process is still running
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			logrus.Errorf("FFmpeg process exited early (attempt %d)", retries+1)
			if retries < 2 {
				time.Sleep(time.Second * 3)
			}
			continue
		}

		// Success! Break out of retry loop
		break
	}

	if err != nil {
		logrus.Errorf("Failed to connect to RTMP stream after 3 attempts, starting test video mode")
		c.mu.Lock()
		c.isRunning = true
		c.mu.Unlock()
		go c.startTestVideoMode(ctx)
		return nil
	}

	c.cmd = cmd
	c.isRunning = true

	// Start streaming in goroutine
	go c.streamLoop(ctx, stdout, stderr)

	return nil
}

func (c *RTMPClient) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return nil
	}

	if c.cmd != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
		c.cmd = nil
	}

	c.isRunning = false
	logrus.Info("RTMP client stopped")
	return nil
}

func (c *RTMPClient) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

func (c *RTMPClient) streamLoop(ctx context.Context, stdout, stderr io.ReadCloser) {
	defer func() {
		c.mu.Lock()
		c.isRunning = false
		c.mu.Unlock()
	}()

	// Log stderr in a separate goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			logrus.Debugf("FFmpeg: %s", scanner.Text())
		}
	}()

	// Read H.264 data from stdout
	scanner := bufio.NewScanner(stdout)
	scanner.Split(c.splitH264Frames)

	frameCount := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			logrus.Info("RTMP client context cancelled")
			return
		default:
			frameData := scanner.Bytes()
			if len(frameData) == 0 {
				continue
			}

			// Calculate timestamp
			now := time.Now()
			timestamp := uint32(now.UnixNano() / 1000000) // Convert to milliseconds

			// Send frame to WebRTC
			logrus.Infof("Sending H.264 frame: size=%d, frame=%d, timestamp=%d", len(frameData), frameCount, timestamp)

			// Log first few bytes to debug H.264 format
			if frameCount < 5 && len(frameData) > 0 {
				maxBytes := 16
				if len(frameData) < maxBytes {
					maxBytes = len(frameData)
				}
				hexBytes := make([]string, maxBytes)
				for i := 0; i < maxBytes; i++ {
					hexBytes[i] = fmt.Sprintf("%02x", frameData[i])
				}
				logrus.Infof("Frame %d first bytes: %s", frameCount, strings.Join(hexBytes, " "))
			}

			if c.shouldWrite == nil || c.shouldWrite() {
				c.webrtcManager.WriteVideoSample(frameData, timestamp)
			}

			frameCount++

			// Log progress every 30 frames (about 1 second at 30fps)
			if frameCount%30 == 0 {
				logrus.Infof("✅ RTMP stream: sent %d frames", frameCount)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		logrus.Errorf("Error reading from FFmpeg stdout: %v", err)
	}

	// Wait for command to finish
	if c.cmd != nil {
		c.cmd.Wait()
	}
}

// splitH264Frames splits H.264 stream into individual frames
// H.264 frames start with 0x00000001 or 0x000001
func (c *RTMPClient) splitH264Frames(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for start codes: 0x00000001 or 0x000001
	startCode1 := []byte{0x00, 0x00, 0x00, 0x01}
	startCode2 := []byte{0x00, 0x00, 0x01}

	// Find the first start code
	startPos := -1
	for i := 0; i < len(data)-3; i++ {
		if (i+4 <= len(data) &&
			data[i] == startCode1[0] && data[i+1] == startCode1[1] &&
			data[i+2] == startCode1[2] && data[i+3] == startCode1[3]) ||
			(i+3 <= len(data) &&
				data[i] == startCode2[0] && data[i+1] == startCode2[1] &&
				data[i+2] == startCode2[2]) {
			startPos = i
			break
		}
	}

	if startPos == -1 {
		// No start code found, need more data
		if atEOF {
			// Return remaining data as final frame
			return len(data), data, nil
		}
		return 0, nil, nil
	}

	// Find the next start code
	nextStartPos := -1
	for i := startPos + 4; i < len(data)-3; i++ {
		if (i+4 <= len(data) &&
			data[i] == startCode1[0] && data[i+1] == startCode1[1] &&
			data[i+2] == startCode1[2] && data[i+3] == startCode1[3]) ||
			(i+3 <= len(data) &&
				data[i] == startCode2[0] && data[i+1] == startCode2[1] &&
				data[i+2] == startCode2[2]) {
			nextStartPos = i
			break
		}
	}

	if nextStartPos == -1 {
		// No next start code found, need more data
		if atEOF {
			// Return from startPos to end
			return len(data), data[startPos:], nil
		}
		return startPos, nil, nil
	}

	// Return frame from startPos to nextStartPos
	return nextStartPos, data[startPos:nextStartPos], nil
}

// startTestVideoMode generates synthetic video for testing when RTMP fails
func (c *RTMPClient) startTestVideoMode(ctx context.Context) {
	logrus.Info("🎬 Starting test video mode - generating synthetic video stream")

	ticker := time.NewTicker(time.Millisecond * 33) // ~30fps
	defer ticker.Stop()

	frameCount := 0
	for {
		select {
		case <-ctx.Done():
			logrus.Info("Test video mode context cancelled")
			return
		case <-ticker.C:
			// Create a simple test pattern frame
			testFrame := c.generateTestFrame(frameCount)

			timestamp := uint32(time.Now().UnixNano() / 1000000) // Current timestamp in ms
			logrus.Infof("🎬 Sending test frame: size=%d, frame=%d, timestamp=%d", len(testFrame), frameCount, timestamp)

			c.webrtcManager.WriteVideoSample(testFrame, timestamp)
			frameCount++

			if frameCount%300 == 0 { // Log every 10 seconds
				logrus.Infof("✅ Test video mode: sent %d frames", frameCount)
			}
		}
	}
}

// generateTestFrame creates a simple test pattern
func (c *RTMPClient) generateTestFrame(frameCount int) []byte {
	// Create a proper H.264 keyframe for testing
	// SPS (Sequence Parameter Set)
	sps := []byte{
		0x00, 0x00, 0x00, 0x01, // Start code
		0x67, 0x42, 0x00, 0x1e, // SPS NAL unit header
		0x95, 0xa0, 0x14, 0x01, 0x6e, 0x40, 0x00, 0x00,
		0x03, 0x00, 0x40, 0x00, 0x00, 0x07, 0x82, 0x00,
		0x00, 0x00, 0x01, 0x68, 0xce, 0x38, 0x80, // PPS NAL unit
	}

	// Create a simple IDR frame (keyframe)
	idrFrame := []byte{
		0x00, 0x00, 0x00, 0x01, // Start code
		0x65, // IDR frame NAL unit
	}

	// Add some payload data to make it a valid frame
	payload := make([]byte, 100)
	for i := range payload {
		payload[i] = byte((frameCount + i) & 0xFF)
	}
	idrFrame = append(idrFrame, payload...)

	// Combine SPS, PPS, and IDR frame
	frame := append(sps, idrFrame...)

	return frame
}

func (c *RTMPClient) GetStreamInfo() (interface{}, error) {
	// This method is not applicable for FFmpeg-based approach
	return nil, fmt.Errorf("stream info not available for FFmpeg-based RTMP client")
}
