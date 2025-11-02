package webrtc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	peers     map[string]*Peer
	peersLock sync.RWMutex
	// RTP packetization state
	rtpSequenceNumber uint16
	rtpTimestamp      uint32
	rtpSSRC           uint32
	// Video timestamp tracking (in 90kHz clock for H.264)
	videoTimestamp     uint32
	videoTimestampLock sync.Mutex
	lastFrameTime      time.Time
	frameRate          float64
	// Real-time snapshot capture
	snapshotRequest chan bool
	snapshotData    chan []byte
	snapshotReady   bool
}

type Peer struct {
	ID          string
	Connection  *webrtc.PeerConnection
	VideoTrack  *webrtc.TrackLocalStaticSample
	AudioTrack  *webrtc.TrackLocalStaticSample
	DataChannel *webrtc.DataChannel
	IsConnected bool
	mu          sync.RWMutex
}

type OfferRequest struct {
	SDP string `json:"sdp"`
}

type OfferResponse struct {
	SDP string `json:"sdp"`
}

func NewManager() *Manager {
	return &Manager{
		peers:             make(map[string]*Peer),
		rtpSequenceNumber: 0,
		rtpTimestamp:      0,
		rtpSSRC:           0x12345678, // Random SSRC
		videoTimestamp:    0,
		lastFrameTime:     time.Now(),
		frameRate:         30.0, // Default 30fps
		snapshotRequest:   make(chan bool, 1),
		snapshotData:      make(chan []byte, 1),
		snapshotReady:     false,
	}
}

func (m *Manager) CreatePeer(peerID string) (*Peer, error) {
	m.peersLock.Lock()
	defer m.peersLock.Unlock()

	// Create WebRTC configuration optimized for local development
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
			{
				URLs: []string{"stun:stun1.l.google.com:19302"},
			},
			{
				URLs: []string{"stun:stun2.l.google.com:19302"},
			},
			{
				URLs: []string{"stun:stun3.l.google.com:19302"},
			},
			{
				URLs: []string{"stun:stun4.l.google.com:19302"},
			},
			// Local TURN server for development
			{
				URLs:       []string{"turn:127.0.0.1:3478"},
				Username:   "webrtc",
				Credential: "webrtc123",
			},
			{
				URLs:       []string{"turn:127.0.0.1:3478"},
				Username:   "test",
				Credential: "test123",
			},
		},
		ICETransportPolicy:   webrtc.ICETransportPolicyAll,
		BundlePolicy:         webrtc.BundlePolicyBalanced,
		RTCPMuxPolicy:        webrtc.RTCPMuxPolicyRequire,
		ICECandidatePoolSize: 10,
	}

	// Create peer connection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	// Create video track - use H.264 for better compatibility with RTMP streams
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeH264,
			ClockRate:    90000,
			Channels:     0,
			SDPFmtpLine:  "profile-level-id=42e01f;packetization-mode=1",
			RTCPFeedback: nil,
		},
		"video",
		"stream",
	)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to create video track: %w", err)
	}

	// Create audio track
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio",
		"stream",
	)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to create audio track: %w", err)
	}

	// Add tracks to peer connection
	if _, err = peerConnection.AddTrack(videoTrack); err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to add video track: %w", err)
	}

	if _, err = peerConnection.AddTrack(audioTrack); err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to add audio track: %w", err)
	}

	// Create data channel for signaling
	dataChannel, err := peerConnection.CreateDataChannel("signaling", nil)
	if err != nil {
		logrus.Warnf("Failed to create data channel: %v", err)
	}

	peer := &Peer{
		ID:          peerID,
		Connection:  peerConnection,
		VideoTrack:  videoTrack,
		AudioTrack:  audioTrack,
		DataChannel: dataChannel,
		IsConnected: false,
	}

	// Set up connection state change handler
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		peer.mu.Lock()
		peer.IsConnected = (state == webrtc.PeerConnectionStateConnected)
		peer.mu.Unlock()

		logrus.Infof("Peer %s connection state: %s", peerID, state.String())

		if state == webrtc.PeerConnectionStateClosed || state == webrtc.PeerConnectionStateFailed {
			m.RemovePeer(peerID)
		}
	})

	// Set up ICE connection state change handler
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		logrus.Infof("Peer %s ICE connection state: %s", peerID, state.String())
	})

	// Set up ICE candidate handler for local development
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			logrus.Infof("Peer %s ICE candidate: %s", peerID, candidate.String())
		} else {
			logrus.Infof("Peer %s ICE gathering complete", peerID)
		}
	})

	// Set up ICE gathering state change handler
	peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		logrus.Infof("Peer %s ICE gathering state: %s", peerID, state.String())
	})

	m.peers[peerID] = peer
	logrus.Infof("Created peer: %s", peerID)

	return peer, nil
}

func (m *Manager) GetPeer(peerID string) (*Peer, bool) {
	m.peersLock.RLock()
	defer m.peersLock.RUnlock()
	peer, exists := m.peers[peerID]
	return peer, exists
}

func (m *Manager) RemovePeer(peerID string) {
	m.peersLock.Lock()
	defer m.peersLock.Unlock()

	if peer, exists := m.peers[peerID]; exists {
		peer.Connection.Close()
		delete(m.peers, peerID)
		logrus.Infof("Removed peer: %s", peerID)
	}
}

func (m *Manager) HandleOffer(peerID string, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	peer, exists := m.GetPeer(peerID)
	if !exists {
		return nil, fmt.Errorf("peer not found: %s", peerID)
	}

	logrus.Infof("Handling offer for peer %s: %+v", peerID, offer)

	// Set remote description
	if err := peer.Connection.SetRemoteDescription(offer); err != nil {
		logrus.Errorf("Failed to set remote description: %v", err)
		return nil, fmt.Errorf("failed to set remote description: %w", err)
	}

	logrus.Infof("Remote description set successfully for peer %s", peerID)

	// Create answer
	answer, err := peer.Connection.CreateAnswer(nil)
	if err != nil {
		logrus.Errorf("Failed to create answer: %v", err)
		return nil, fmt.Errorf("failed to create answer: %w", err)
	}

	logrus.Infof("Answer created successfully for peer %s", peerID)

	// Set local description
	if err := peer.Connection.SetLocalDescription(answer); err != nil {
		logrus.Errorf("Failed to set local description: %v", err)
		return nil, fmt.Errorf("failed to set local description: %w", err)
	}

	logrus.Infof("Local description set successfully for peer %s", peerID)

	// Wait for ICE gathering to complete so the client receives a full, non-trickle SDP
	iceComplete := webrtc.GatheringCompletePromise(peer.Connection)
	<-iceComplete
	local := peer.Connection.LocalDescription()

	// Mark peer as connected after successful SDP negotiation
	peer.mu.Lock()
	peer.IsConnected = true
	peer.mu.Unlock()
	logrus.Infof("Peer %s marked as connected after SDP negotiation", peerID)

	return local, nil
}

func (m *Manager) WriteVideoSample(data []byte, timestamp uint32) {
	if len(data) == 0 {
		return
	}

	// Check if snapshot is requested and capture this frame
	select {
	case <-m.snapshotRequest:
		// Capture this frame for snapshot
		frameCopy := make([]byte, len(data))
		copy(frameCopy, data)
		select {
		case m.snapshotData <- frameCopy:
			logrus.Info("Frame captured for snapshot")
		default:
			// Channel is full, skip this frame
			logrus.Warn("Snapshot channel full, skipping frame")
		}
	default:
		// No snapshot request, continue normally
	}

	// Calculate proper timestamp in 90kHz clock (H.264 standard)
	// 90kHz = 90,000 ticks per second = 90,000,000 ticks per millisecond
	m.videoTimestampLock.Lock()
	now := time.Now()
	if m.lastFrameTime.IsZero() {
		m.lastFrameTime = now
		m.videoTimestamp = 0
	} else {
		// Calculate time delta and increment timestamp accordingly
		elapsed := now.Sub(m.lastFrameTime)
		// Convert elapsed time to 90kHz ticks: elapsed_ns * 90,000 / 1,000,000,000
		// For better precision, use: elapsed_ns / 1,000,000,000 * 90,000
		elapsedNs := elapsed.Nanoseconds()
		// Use integer math: multiply by 90,000 first, then divide by 1 billion
		timestampDelta := uint32(elapsedNs * 90000 / 1000000000)

		// Ensure minimum timestamp increment (avoid 0 delta which causes issues)
		if timestampDelta == 0 {
			// Default to ~33.33ms at 30fps = 3000 ticks at 90kHz
			timestampDelta = 3000
		}
		// Cap maximum delta to prevent large jumps (max 100ms = 9000 ticks)
		if timestampDelta > 9000 {
			timestampDelta = 3000 // Reset to normal frame interval
		}

		m.videoTimestamp += timestampDelta
		m.lastFrameTime = now
	}
	currentTimestamp := m.videoTimestamp
	m.videoTimestampLock.Unlock()

	// Parse H.264 NAL units from the data
	nalUnits, err := m.parseH264NALUnits(data)
	if err != nil {
		logrus.Errorf("Failed to parse H.264 NAL units: %v", err)
		return
	}

	if len(nalUnits) == 0 {
		return
	}

	// Calculate frame duration (for 30fps = 33.33ms = 3000 ticks at 90kHz)
	frameDuration := time.Duration(float64(time.Second) / m.frameRate)

	// Separate SPS/PPS from other NAL units
	var spsPpsUnits [][]byte
	var frameUnits [][]byte

	for _, nalUnit := range nalUnits {
		if len(nalUnit) == 0 {
			continue
		}
		nalType := nalUnit[0] & 0x1F
		if nalType == 7 || nalType == 8 { // SPS or PPS
			spsPpsUnits = append(spsPpsUnits, nalUnit)
		} else {
			frameUnits = append(frameUnits, nalUnit)
		}
	}

	m.peersLock.RLock()
	peers := make([]*Peer, 0, len(m.peers))
	for _, peer := range m.peers {
		peers = append(peers, peer)
	}
	m.peersLock.RUnlock()

	// Send SPS/PPS first (they don't need timestamp increment)
	for _, peer := range peers {
		peer.mu.RLock()
		hasVideoTrack := peer.VideoTrack != nil
		peer.mu.RUnlock()

		if !hasVideoTrack {
			continue
		}

		for _, nalUnit := range spsPpsUnits {
			sample := media.Sample{
				Data:            nalUnit,
				Duration:        0, // SPS/PPS have no duration
				PacketTimestamp: 0, // Use 0 for parameter sets
			}

			if err := peer.VideoTrack.WriteSample(sample); err != nil {
				logrus.Errorf("Failed to write SPS/PPS to peer %s: %v", peer.ID, err)
			}
		}
	}

	// Send frame NAL units with proper timestamp
	for _, peer := range peers {
		peer.mu.RLock()
		hasVideoTrack := peer.VideoTrack != nil
		peer.mu.RUnlock()

		if !hasVideoTrack {
			continue
		}

		// Send all NAL units from the same frame with the same timestamp
		for _, nalUnit := range frameUnits {
			sample := media.Sample{
				Data:            nalUnit,
				Duration:        frameDuration,
				PacketTimestamp: currentTimestamp,
			}

			if err := peer.VideoTrack.WriteSample(sample); err != nil {
				logrus.Errorf("Failed to write video sample to peer %s: %v", peer.ID, err)
			}
		}
	}
}

func (m *Manager) WriteAudioSample(data []byte, timestamp uint32) {
	m.peersLock.RLock()
	defer m.peersLock.RUnlock()

	for _, peer := range m.peers {
		peer.mu.RLock()
		if peer.IsConnected && peer.AudioTrack != nil {
			sample := media.Sample{
				Data:     data,
				Duration: time.Millisecond * 20, // ~50fps for audio
			}
			if timestamp > 0 {
				sample.PacketTimestamp = timestamp
			}
			if err := peer.AudioTrack.WriteSample(sample); err != nil {
				logrus.Errorf("Failed to write audio sample to peer %s: %v", peer.ID, err)
			}
		}
		peer.mu.RUnlock()
	}
}

func (m *Manager) GetConnectedPeersCount() int {
	m.peersLock.RLock()
	defer m.peersLock.RUnlock()

	count := 0
	for _, peer := range m.peers {
		peer.mu.RLock()
		if peer.IsConnected {
			count++
		}
		peer.mu.RUnlock()
	}
	return count
}

func (m *Manager) GetAllPeers() map[string]*Peer {
	m.peersLock.RLock()
	defer m.peersLock.RUnlock()

	// Return a copy to avoid race conditions
	peers := make(map[string]*Peer)
	for id, peer := range m.peers {
		peers[id] = peer
	}
	return peers
}

// parseH264NALUnits extracts NAL units from H.264 data
func (m *Manager) parseH264NALUnits(data []byte) ([][]byte, error) {
	var nalUnits [][]byte

	// Look for start codes: 0x00000001 or 0x000001
	startCode1 := []byte{0x00, 0x00, 0x00, 0x01}
	startCode2 := []byte{0x00, 0x00, 0x01}

	offset := 0
	for offset < len(data) {
		// Find next start code
		startPos := -1
		for i := offset; i < len(data)-3; i++ {
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
			// No more start codes found
			break
		}

		// Skip the start code
		nalStart := startPos
		if data[startPos+3] == 0x01 {
			nalStart += 4 // 0x00000001
		} else {
			nalStart += 3 // 0x000001
		}

		// Find next start code
		nextStartPos := -1
		for i := nalStart; i < len(data)-3; i++ {
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
			// Last NAL unit
			nalUnits = append(nalUnits, data[nalStart:])
			break
		} else {
			nalUnits = append(nalUnits, data[nalStart:nextStartPos])
			offset = nextStartPos
		}
	}

	return nalUnits, nil
}

// packetizeNALUnit converts a NAL unit to RTP packets
func (m *Manager) packetizeNALUnit(nalUnit []byte, timestamp uint32) [][]byte {
	if len(nalUnit) == 0 {
		return nil
	}

	// RTP header size
	const rtpHeaderSize = 12
	const maxPayloadSize = 1400 // MTU - IP - UDP - RTP headers

	// Update RTP timestamp
	if timestamp > 0 {
		m.rtpTimestamp = timestamp * 90 // Convert ms to 90kHz clock
	} else {
		m.rtpTimestamp += 3000 // ~33ms at 90kHz
	}

	// Increment sequence number
	m.rtpSequenceNumber++

	// If NAL unit fits in one packet
	if len(nalUnit) <= maxPayloadSize {
		rtpPacket := make([]byte, rtpHeaderSize+len(nalUnit))

		// RTP header
		rtpPacket[0] = 0x80 // Version 2, no padding, no extension, no CSRC
		rtpPacket[1] = 0x60 // Payload type 96 (H.264)
		rtpPacket[2] = byte(m.rtpSequenceNumber >> 8)
		rtpPacket[3] = byte(m.rtpSequenceNumber)
		rtpPacket[4] = byte(m.rtpTimestamp >> 24)
		rtpPacket[5] = byte(m.rtpTimestamp >> 16)
		rtpPacket[6] = byte(m.rtpTimestamp >> 8)
		rtpPacket[7] = byte(m.rtpTimestamp)
		rtpPacket[8] = byte(m.rtpSSRC >> 24)
		rtpPacket[9] = byte(m.rtpSSRC >> 16)
		rtpPacket[10] = byte(m.rtpSSRC >> 8)
		rtpPacket[11] = byte(m.rtpSSRC)

		// Copy NAL unit
		copy(rtpPacket[rtpHeaderSize:], nalUnit)

		return [][]byte{rtpPacket}
	}

	// Fragment the NAL unit using FU-A
	var packets [][]byte
	nalType := nalUnit[0] & 0x1F
	nalHeader := (nalUnit[0] & 0x60) | 28 // FU-A type

	offset := 1 // Skip NAL header
	for offset < len(nalUnit) {
		payloadSize := maxPayloadSize - 2 // Reserve space for FU-A header
		if offset+payloadSize > len(nalUnit) {
			payloadSize = len(nalUnit) - offset
		}

		rtpPacket := make([]byte, rtpHeaderSize+2+payloadSize)

		// RTP header
		rtpPacket[0] = 0x80
		rtpPacket[1] = 0x60
		rtpPacket[2] = byte(m.rtpSequenceNumber >> 8)
		rtpPacket[3] = byte(m.rtpSequenceNumber)
		rtpPacket[4] = byte(m.rtpTimestamp >> 24)
		rtpPacket[5] = byte(m.rtpTimestamp >> 16)
		rtpPacket[6] = byte(m.rtpTimestamp >> 8)
		rtpPacket[7] = byte(m.rtpTimestamp)
		rtpPacket[8] = byte(m.rtpSSRC >> 24)
		rtpPacket[9] = byte(m.rtpSSRC >> 16)
		rtpPacket[10] = byte(m.rtpSSRC >> 8)
		rtpPacket[11] = byte(m.rtpSSRC)

		// FU-A header
		if offset == 1 {
			rtpPacket[12] = nalHeader | 0x80 // Start bit
		} else if offset+payloadSize >= len(nalUnit) {
			rtpPacket[12] = nalHeader | 0x40 // End bit
		} else {
			rtpPacket[12] = nalHeader
		}
		rtpPacket[13] = nalType

		// Copy payload
		copy(rtpPacket[rtpHeaderSize+2:], nalUnit[offset:offset+payloadSize])

		packets = append(packets, rtpPacket)
		offset += payloadSize
		m.rtpSequenceNumber++
	}

	return packets
}

// addH264StartCode adds H.264 start code to raw NAL unit data
func (m *Manager) addH264StartCode(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	// Check if start code already exists
	if len(data) >= 4 && data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x00 && data[3] == 0x01 {
		return data // Already has start code
	}
	if len(data) >= 3 && data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 {
		return data // Already has short start code
	}

	// Add start code
	startCode := []byte{0x00, 0x00, 0x00, 0x01}
	return append(startCode, data...)
}

// createRTPPacket creates a simple RTP packet for H.264
func (m *Manager) createRTPPacket(nalUnit []byte, timestamp uint32) []byte {
	if len(nalUnit) == 0 {
		return nil
	}

	// RTP header size
	const rtpHeaderSize = 12

	// Update RTP timestamp
	if timestamp > 0 {
		m.rtpTimestamp = timestamp * 90 // Convert ms to 90kHz clock
	} else {
		m.rtpTimestamp += 3000 // ~33ms at 90kHz
	}

	// Increment sequence number
	m.rtpSequenceNumber++

	rtpPacket := make([]byte, rtpHeaderSize+len(nalUnit))

	// RTP header
	rtpPacket[0] = 0x80 // Version 2, no padding, no extension, no CSRC
	rtpPacket[1] = 0x60 // Payload type 96 (H.264)
	rtpPacket[2] = byte(m.rtpSequenceNumber >> 8)
	rtpPacket[3] = byte(m.rtpSequenceNumber)
	rtpPacket[4] = byte(m.rtpTimestamp >> 24)
	rtpPacket[5] = byte(m.rtpTimestamp >> 16)
	rtpPacket[6] = byte(m.rtpTimestamp >> 8)
	rtpPacket[7] = byte(m.rtpTimestamp)
	rtpPacket[8] = byte(m.rtpSSRC >> 24)
	rtpPacket[9] = byte(m.rtpSSRC >> 16)
	rtpPacket[10] = byte(m.rtpSSRC >> 8)
	rtpPacket[11] = byte(m.rtpSSRC)

	// Copy NAL unit
	copy(rtpPacket[rtpHeaderSize:], nalUnit)

	return rtpPacket
}

// RequestSnapshot triggers a snapshot capture from the next available video frame
func (m *Manager) RequestSnapshot() {
	select {
	case m.snapshotRequest <- true:
		logrus.Info("Snapshot request sent")
	default:
		logrus.Warn("Snapshot request channel full")
	}
}

// CaptureSnapshot captures a frame from the live stream and converts it to JPEG
func (m *Manager) CaptureSnapshot() (string, error) {
	// Request a snapshot from the live stream
	m.RequestSnapshot()

	// Wait for the next frame to be captured (with timeout)
	select {
	case frameData := <-m.snapshotData:
		if len(frameData) == 0 {
			return "", fmt.Errorf("empty frame received")
		}

		logrus.Infof("Captured frame for snapshot: %d bytes", len(frameData))

		// Convert H.264 frame to JPEG
		jpegData, err := m.convertH264ToJPEG(frameData)
		if err != nil {
			return "", fmt.Errorf("failed to convert H.264 to JPEG: %w", err)
		}

		// Encode to base64
		base64Data := base64.StdEncoding.EncodeToString(jpegData)
		return "data:image/jpeg;base64," + base64Data, nil

	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("timeout waiting for video frame")
	}
}

// convertH264ToJPEG converts H.264 frame to JPEG using FFmpeg
func (m *Manager) convertH264ToJPEG(h264Data []byte) ([]byte, error) {
	// Check if FFmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		logrus.Warnf("FFmpeg not found, using placeholder image: %v", err)
		return m.createPlaceholderJPEG()
	}

	// Create temporary files for input and output
	inputFile, err := os.CreateTemp("", "h264_input_*.h264")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", err)
	}
	defer os.Remove(inputFile.Name())
	defer inputFile.Close()

	outputFile, err := os.CreateTemp("", "jpeg_output_*.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	defer os.Remove(outputFile.Name())
	defer outputFile.Close()

	// Write H.264 data to input file
	if _, err := inputFile.Write(h264Data); err != nil {
		return nil, fmt.Errorf("failed to write H.264 data: %w", err)
	}
	inputFile.Close()
	outputFile.Close()

	// Run FFmpeg to convert H.264 to JPEG
	cmd := exec.Command("ffmpeg",
		"-i", inputFile.Name(),
		"-vframes", "1",
		"-f", "image2",
		"-y", // Overwrite output file
		outputFile.Name(),
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logrus.Errorf("FFmpeg conversion failed: %v, stderr: %s", err, stderr.String())
		// Fallback to placeholder if FFmpeg fails
		return m.createPlaceholderJPEG()
	}

	// Read the output JPEG file
	jpegData, err := os.ReadFile(outputFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read output JPEG file: %w", err)
	}

	return jpegData, nil
}

// createPlaceholderJPEG creates a simple placeholder JPEG image
func (m *Manager) createPlaceholderJPEG() ([]byte, error) {
	// Create a simple 100x100 red square JPEG as placeholder
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
