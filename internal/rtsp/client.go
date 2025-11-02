package rtsp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	webrtcmanager "golang-webrtc-streaming/internal/webrtc"

	"github.com/sirupsen/logrus"
)

type Client struct {
	url           string
	webrtcManager *webrtcmanager.Manager
	cmd           *exec.Cmd
	isRunning     bool
	mu            sync.RWMutex
	shouldWrite   func() bool
}

func NewClient(rtspURL string, webrtcManager *webrtcmanager.Manager, shouldWrite func() bool) *Client {
	return &Client{
		url:           rtspURL,
		webrtcManager: webrtcManager,
		shouldWrite:   shouldWrite,
	}
}

func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return fmt.Errorf("RTSP client is already running")
	}
	c.isRunning = true
	c.mu.Unlock()

	logrus.Infof("Starting RTSP client supervisor for: %s", c.url)

	go c.supervise(ctx)
	return nil
}

func (c *Client) supervise(ctx context.Context) {
	backoff := time.Second * 2
	const maxBackoff = time.Second * 20

	for {
		select {
		case <-ctx.Done():
			c.setRunning(false)
			return
		default:
		}

		// Run one ffmpeg session
		err := c.runOnce(ctx)
		if err != nil {
			logrus.Errorf("RTSP pipeline error: %v", err)
		}

		// Backoff before restarting
		logrus.Infof("RTSP restarting in %s...", backoff)
		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (c *Client) runOnce(ctx context.Context) error {
	logrus.Infof("Starting RTSP ffmpeg for: %s", c.url)

	transport := os.Getenv("RTSP_TRANSPORT")
	if transport == "" {
		transport = "tcp"
	}

	// Force transcode to H.264 to handle non-H264 cameras reliably
	// Handle both HEVC and H.264 input streams
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-rtsp_transport", transport,
		"-fflags", "+genpts", // Generate presentation timestamps
		"-avoid_negative_ts", "make_zero", // Handle negative timestamps
		"-i", c.url,
		"-an",             // No audio
		"-c:v", "libx264", // Use H.264 encoder
		"-preset", "veryfast", // Fast encoding
		"-tune", "zerolatency", // Optimize for low latency
		"-profile:v", "baseline", // Use baseline profile for compatibility
		"-level", "3.1", // Level 3.1 for compatibility
		"-pix_fmt", "yuv420p", // Pixel format
		"-g", "30", // GOP size for better compatibility
		"-keyint_min", "30", // Minimum keyframe interval
		"-sc_threshold", "0", // Disable scene change detection
		"-bf", "0", // No B-frames for lower latency
		"-flags", "+low_delay", // Low delay flags
		"-f", "h264", // Output format
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	c.setCmd(cmd)
	logrus.Infof("FFmpeg process started with PID: %d", cmd.Process.Pid)

	// Stream loop blocks until EOF or error
	c.streamLoop(ctx, stdout, stderr)

	// Ensure process exited
	if err := cmd.Wait(); err != nil {
		logrus.Warnf("FFmpeg process exited with error: %v", err)
	} else {
		logrus.Info("FFmpeg process exited normally")
	}
	c.clearCmd()

	return nil
}

func (c *Client) setCmd(cmd *exec.Cmd) {
	c.mu.Lock()
	c.cmd = cmd
	c.mu.Unlock()
}

func (c *Client) clearCmd() {
	c.mu.Lock()
	c.cmd = nil
	c.mu.Unlock()
}

func (c *Client) setRunning(v bool) {
	c.mu.Lock()
	c.isRunning = v
	c.mu.Unlock()
}

func (c *Client) Stop() error {
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
	logrus.Info("RTSP client stopped")
	return nil
}

func (c *Client) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

func (c *Client) streamLoop(ctx context.Context, stdout, stderr io.ReadCloser) {
	// mark running for this session
	c.setRunning(true)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			// Log errors and warnings more prominently
			if strings.Contains(line, "error") || strings.Contains(line, "Error") ||
				strings.Contains(line, "failed") || strings.Contains(line, "Failed") ||
				strings.Contains(line, "warning") || strings.Contains(line, "Warning") {
				logrus.Warnf("FFmpeg (rtsp): %s", line)
			} else {
				logrus.Debugf("FFmpeg (rtsp): %s", line)
			}
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Split(splitH264Frames)

	frameCount := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			logrus.Info("RTSP client context cancelled")
			return
		default:
			frameData := scanner.Bytes()
			if len(frameData) == 0 {
				continue
			}

			// Timestamp is now handled in WebRTC manager
			timestamp := uint32(0)

			// Only log first frame for debugging
			if frameCount == 0 && len(frameData) > 0 {
				maxBytes := 16
				if len(frameData) < maxBytes {
					maxBytes = len(frameData)
				}
				hexBytes := make([]string, maxBytes)
				for i := 0; i < maxBytes; i++ {
					hexBytes[i] = fmt.Sprintf("%02x", frameData[i])
				}
				logrus.Infof("RTSP: First frame bytes: %s (size: %d)", strings.Join(hexBytes, " "), len(frameData))
			}

			if c.shouldWrite == nil || c.shouldWrite() {
				c.webrtcManager.WriteVideoSample(frameData, timestamp)
			}
			frameCount++

			// Log progress every 300 frames (~10 seconds at 30fps) instead of every 30
			if frameCount%300 == 0 {
				logrus.Infof("✅ RTSP stream: sent %d frames", frameCount)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		logrus.Errorf("Error reading from FFmpeg stdout (rtsp): %v", err)
	}

	c.setRunning(false)
}

// splitH264Frames splits an H.264 bytestream into NAL units delimited by start codes
func splitH264Frames(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	startCode1 := []byte{0x00, 0x00, 0x00, 0x01}
	startCode2 := []byte{0x00, 0x00, 0x01}

	startPos := -1
	for i := 0; i < len(data)-3; i++ {
		if (i+4 <= len(data) && data[i] == startCode1[0] && data[i+1] == startCode1[1] && data[i+2] == startCode1[2] && data[i+3] == startCode1[3]) ||
			(i+3 <= len(data) && data[i] == startCode2[0] && data[i+1] == startCode2[1] && data[i+2] == startCode2[2]) {
			startPos = i
			break
		}
	}

	if startPos == -1 {
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}

	nextStartPos := -1
	for i := startPos + 4; i < len(data)-3; i++ {
		if (i+4 <= len(data) && data[i] == startCode1[0] && data[i+1] == startCode1[1] && data[i+2] == startCode1[2] && data[i+3] == startCode1[3]) ||
			(i+3 <= len(data) && data[i] == startCode2[0] && data[i+1] == startCode2[1] && data[i+2] == startCode2[2]) {
			nextStartPos = i
			break
		}
	}

	if nextStartPos == -1 {
		if atEOF {
			return len(data), data[startPos:], nil
		}
		return startPos, nil, nil
	}

	return nextStartPos, data[startPos:nextStartPos], nil
}
