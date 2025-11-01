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

	// MediaMTX already provides optimized H.264 streams
	// Just copy the stream without transcoding for lowest latency
	// Add timeout and connection retry flags
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-rtsp_transport", transport,
		"-fflags", "+genpts", // Generate presentation timestamps
		"-avoid_negative_ts", "make_zero", // Handle negative timestamps
		"-stimeout", "5000000", // 5 second socket timeout (in microseconds)
		"-i", c.url,
		"-an",       // No audio
		"-c:v", "copy", // Copy video stream (no transcoding - MediaMTX handles optimization)
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
			lineLower := strings.ToLower(line)
			if strings.Contains(lineLower, "error") ||
				strings.Contains(lineLower, "failed") ||
				strings.Contains(lineLower, "cannot connect") ||
				strings.Contains(lineLower, "connection refused") ||
				strings.Contains(lineLower, "connection timed out") ||
				strings.Contains(lineLower, "unable to open") {
				logrus.Errorf("FFmpeg (rtsp) ERROR: %s", line)
			} else if strings.Contains(lineLower, "warning") {
				logrus.Warnf("FFmpeg (rtsp): %s", line)
			} else if strings.Contains(lineLower, "stream") || strings.Contains(lineLower, "frame") {
				logrus.Debugf("FFmpeg (rtsp): %s", line)
			}
		}
		if err := scanner.Err(); err != nil {
			logrus.Errorf("FFmpeg stderr scanner error: %v", err)
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

			timestamp := uint32(time.Now().UnixNano() / 1000000)
			if frameCount < 10 && len(frameData) > 0 {
				maxBytes := 16
				if len(frameData) < maxBytes {
					maxBytes = len(frameData)
				}
				hexBytes := make([]string, maxBytes)
				for i := 0; i < maxBytes; i++ {
					hexBytes[i] = fmt.Sprintf("%02x", frameData[i])
				}
				logrus.Infof("RTSP frame %d first bytes: %s (size: %d)", frameCount, strings.Join(hexBytes, " "), len(frameData))

				// Check for valid H.264 start codes
				if len(frameData) >= 4 {
					startCode1 := frameData[0] == 0x00 && frameData[1] == 0x00 && frameData[2] == 0x00 && frameData[3] == 0x01
					startCode2 := frameData[0] == 0x00 && frameData[1] == 0x00 && frameData[2] == 0x01
					if startCode1 || startCode2 {
						logrus.Infof("RTSP frame %d: Valid H.264 start code detected", frameCount)
					} else {
						logrus.Warnf("RTSP frame %d: No valid H.264 start code found", frameCount)
					}
				}
			}

			if c.shouldWrite == nil || c.shouldWrite() {
				c.webrtcManager.WriteVideoSample(frameData, timestamp)
			}
			frameCount++
			if frameCount%30 == 0 {
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
