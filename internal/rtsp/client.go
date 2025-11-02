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
	// Optimized for low latency streaming with RTSP compatibility
	// Added HEVC decoder options to handle RPS (Reference Picture Set) errors gracefully
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-rtsp_transport", transport,
		"-rtsp_flags", "prefer_tcp", // Prefer TCP for stability
		"-fflags", "+genpts+discardcorrupt", // Generate PTS and discard corrupted frames
		"-flags", "low_delay", // Low delay flag
		"-err_detect", "ignore_err", // Ignore decoder errors (handles HEVC RPS errors)
		"-i", c.url,
		"-an",             // No audio
		"-c:v", "libx264", // Use H.264 encoder
		"-preset", "ultrafast", // Fastest encoding preset
		"-tune", "zerolatency", // Optimize for zero latency
		"-profile:v", "baseline", // Use baseline profile for compatibility
		"-level", "3.1", // Level 3.1 for compatibility
		"-pix_fmt", "yuv420p", // Pixel format
		"-g", "15", // GOP size (balanced for low latency)
		"-keyint_min", "15", // Minimum keyframe interval
		"-sc_threshold", "0", // Disable scene change detection
		"-bf", "0", // No B-frames for lower latency
		"-slices", "1", // Single slice for lower latency
		"-threads", "2", // Allow 2 threads for better performance
		"-b:v", "2M", // Bitrate
		"-maxrate", "2M", // Max bitrate
		"-bufsize", "2M", // Buffer size
		"-vsync", "0", // Passthrough timestamps, avoid frame rate conversion issues
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
	// stderrBuffer will be captured in streamLoop closure
	c.streamLoop(ctx, stdout, stderr)

	// Ensure process exited
	err = cmd.Wait()
	if err != nil {
		logrus.Errorf("FFmpeg process exited with error: %v", err)
		return fmt.Errorf("ffmpeg exited with error: %w", err)
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

	// Capture stderr for error detection and logging
	go func() {
		scanner := bufio.NewScanner(stderr)
		// Increase buffer size to handle long error messages (default is 64KB)
		buf := make([]byte, 0, 1024*1024) // 1MB buffer
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()

			// Log errors and warnings more prominently
			lowerLine := strings.ToLower(line)
			// HEVC RPS (Reference Picture Set) errors are handled gracefully, log as warnings
			isRPSError := strings.Contains(lowerLine, "error constructing the frame rps") ||
				strings.Contains(lowerLine, "error constructing the frame") ||
				strings.Contains(lowerLine, "rps")

			if isRPSError {
				// RPS errors are expected with HEVC streams and are now handled - log as debug
				logrus.Debugf("FFmpeg (rtsp) HEVC decoder: %s", line)
			} else if strings.Contains(lowerLine, "error") ||
				strings.Contains(lowerLine, "failed") ||
				strings.Contains(lowerLine, "unable") ||
				strings.Contains(lowerLine, "connection") ||
				strings.Contains(lowerLine, "timeout") {
				logrus.Errorf("FFmpeg (rtsp) ERROR: %s", line)
			} else if strings.Contains(lowerLine, "warning") {
				logrus.Warnf("FFmpeg (rtsp): %s", line)
			} else {
				// Only log important info lines (stream info, codec, etc.)
				if strings.Contains(line, "Stream") || strings.Contains(line, "codec") ||
					strings.Contains(line, "fps") || strings.Contains(line, "bitrate") {
					logrus.Infof("FFmpeg (rtsp): %s", line)
				} else {
					logrus.Debugf("FFmpeg (rtsp): %s", line)
				}
			}
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Split(splitH264Frames)
	// Increase buffer size to handle large H.264 frames (default is 64KB)
	// H.264 frames can be much larger, especially for high resolution streams
	buf := make([]byte, 0, 10*1024*1024) // 10MB initial capacity
	scanner.Buffer(buf, 10*1024*1024)    // 10MB max token size

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
