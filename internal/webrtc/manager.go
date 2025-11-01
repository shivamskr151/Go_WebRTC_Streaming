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
		snapshotRequest:   make(chan bool, 1),
		snapshotData:      make(chan []byte, 1),
		snapshotReady:     false,
	}
}

func (m *Manager) CreatePeer(peerID string) (*Peer, error) {
	m.peersLock.Lock()
	defer m.peersLock.Unlock()

	// Create WebRTC configuration optimized for connection establishment
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			// Use Google STUN for NAT traversal
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
			{
				URLs: []string{"stun:stun1.l.google.com:19302"},
			},
		},
		ICETransportPolicy:   webrtc.ICETransportPolicyAll, // Allow all candidate types
		BundlePolicy:         webrtc.BundlePolicyMaxCompat, // More compatible
		RTCPMuxPolicy:        webrtc.RTCPMuxPolicyRequire,
		ICECandidatePoolSize: 0, // Disable candidate pool for faster connection
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

	// Data channel not needed for media-only connections
	// Removed to avoid potential connection issues

	peer := &Peer{
		ID:          peerID,
		Connection:  peerConnection,
		VideoTrack:  videoTrack,
		AudioTrack:  audioTrack,
		IsConnected: false,
	}

	// Set up connection state change handler with detailed logging
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		iceState := peerConnection.ICEConnectionState()
		logrus.Infof("Peer %s connection state: %s (ICE: %s)", peerID, state.String(), iceState.String())

		peer.mu.Lock()
		// Only mark as connected if both connection and ICE are established
		peer.IsConnected = (state == webrtc.PeerConnectionStateConnected) &&
			(iceState == webrtc.ICEConnectionStateConnected || iceState == webrtc.ICEConnectionStateCompleted)
		isConnected := peer.IsConnected
		peer.mu.Unlock()

		switch state {
		case webrtc.PeerConnectionStateConnected:
			logrus.Infof("✅ Peer %s PeerConnection is CONNECTED (ICE: %s)", peerID, iceState.String())
			if isConnected {
				logrus.Infof("🎉 Peer %s is fully connected and ready for media!", peerID)
			}
		case webrtc.PeerConnectionStateConnecting:
			logrus.Infof("🔄 Peer %s PeerConnection is CONNECTING (ICE: %s)", peerID, iceState.String())
		case webrtc.PeerConnectionStateDisconnected:
			logrus.Warnf("⚠️ Peer %s PeerConnection DISCONNECTED (ICE: %s)", peerID, iceState.String())
			// Give it time to recover - don't remove immediately
		case webrtc.PeerConnectionStateFailed:
			logrus.Errorf("❌ Peer %s PeerConnection FAILED (ICE: %s)", peerID, iceState.String())
			// Remove peer on failure after a delay
			go func() {
				time.Sleep(5 * time.Second)
				if peer.Connection != nil &&
					peer.Connection.ConnectionState() == webrtc.PeerConnectionStateFailed {
					logrus.Warnf("Removing failed peer %s", peerID)
					m.RemovePeer(peerID)
				}
			}()
		case webrtc.PeerConnectionStateClosed:
			logrus.Infof("🔒 Peer %s PeerConnection CLOSED", peerID)
			m.RemovePeer(peerID)
		case webrtc.PeerConnectionStateNew:
			logrus.Infof("🆕 Peer %s PeerConnection NEW", peerID)
		}
	})

	// Set up ICE connection state change handler
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		logrus.Infof("Peer %s ICE connection state: %s", peerID, state.String())

		// Handle connection states
		switch state {
		case webrtc.ICEConnectionStateConnected:
			logrus.Infof("✅ Peer %s ICE connection established", peerID)
			peer.mu.Lock()
			peer.IsConnected = true
			peer.mu.Unlock()
		case webrtc.ICEConnectionStateCompleted:
			logrus.Infof("✅ Peer %s ICE connection completed", peerID)
			peer.mu.Lock()
			peer.IsConnected = true
			peer.mu.Unlock()
		case webrtc.ICEConnectionStateDisconnected:
			logrus.Warnf("⚠️ Peer %s ICE connection disconnected - may recover", peerID)
			peer.mu.Lock()
			peer.IsConnected = false
			peer.mu.Unlock()

			// Log connection state for debugging
			if peer.Connection != nil {
				logrus.Warnf("Peer %s states: PeerConnection=%s, ICE=%s",
					peerID,
					peer.Connection.ConnectionState().String(),
					state.String())
			}

			// Don't remove peer immediately - give it time to recover
			// ICE disconnection can be transient
			go func() {
				time.Sleep(5 * time.Second)
				if peer.Connection != nil {
					currentICEState := peer.Connection.ICEConnectionState()
					if currentICEState == webrtc.ICEConnectionStateDisconnected ||
						currentICEState == webrtc.ICEConnectionStateFailed {
						logrus.Warnf("Peer %s still disconnected after 5s, state=%s", peerID, currentICEState.String())
					}
				}
			}()
		case webrtc.ICEConnectionStateFailed:
			logrus.Errorf("❌ Peer %s ICE connection failed", peerID)
			peer.mu.Lock()
			peer.IsConnected = false
			peer.mu.Unlock()
			// Attempt ICE restart
			go func() {
				time.Sleep(2 * time.Second)
				if peer.Connection != nil && peer.Connection.ConnectionState() != webrtc.PeerConnectionStateClosed {
					logrus.Infof("Attempting ICE restart for peer %s", peerID)
					if offer, err := peer.Connection.CreateOffer(&webrtc.OfferOptions{ICERestart: true}); err == nil {
						if err := peer.Connection.SetLocalDescription(offer); err == nil {
							logrus.Infof("ICE restart offer created for peer %s", peerID)
						}
					}
				}
			}()
		case webrtc.ICEConnectionStateChecking:
			logrus.Infof("🔍 Peer %s ICE connection checking...", peerID)
		case webrtc.ICEConnectionStateNew:
			logrus.Infof("🆕 Peer %s ICE connection new", peerID)
		case webrtc.ICEConnectionStateClosed:
			logrus.Infof("🔒 Peer %s ICE connection closed", peerID)
			m.RemovePeer(peerID)
		}
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
	// Use a timeout to avoid hanging if ICE gathering fails
	iceComplete := webrtc.GatheringCompletePromise(peer.Connection)
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()

	select {
	case <-iceComplete:
		logrus.Infof("ICE gathering completed for peer %s", peerID)
	case <-timeout.C:
		logrus.Warnf("ICE gathering timeout for peer %s, proceeding with current candidates", peerID)
	}

	local := peer.Connection.LocalDescription()
	if local == nil {
		return nil, fmt.Errorf("local description is nil after ICE gathering")
	}

	// Don't mark as connected here - wait for actual ICE connection
	// The connection state will be updated by the ICE connection state change handler
	logrus.Infof("✅ SDP negotiation complete for peer %s, waiting for ICE connection...", peerID)

	return local, nil
}

func (m *Manager) WriteVideoSample(data []byte, timestamp uint32) {
	if len(data) == 0 {
		logrus.Debugf("Empty video sample received")
		return
	}

	m.peersLock.RLock()
	peerCount := len(m.peers)
	connectedPeers := 0
	for _, peer := range m.peers {
		peer.mu.RLock()
		if peer.Connection != nil &&
			(peer.Connection.ConnectionState() == webrtc.PeerConnectionStateConnected ||
				peer.Connection.ConnectionState() == webrtc.PeerConnectionStateConnecting) {
			connectedPeers++
		}
		peer.mu.RUnlock()
	}
	m.peersLock.RUnlock()

	if peerCount == 0 {
		// No peers connected, nothing to do
		logrus.Debugf("No peers connected, dropping video sample (size: %d)", len(data))
		return
	}

	if connectedPeers == 0 {
		logrus.Debugf("No connected peers, dropping video sample (size: %d, total peers: %d)", len(data), peerCount)
		return
	}

	logrus.Debugf("📹 Writing video sample: size=%d, timestamp=%d, total_peers=%d, connected_peers=%d",
		len(data), timestamp, peerCount, connectedPeers)

	m.peersLock.RLock()
	defer m.peersLock.RUnlock()

	// Check if data has valid H.264 start codes
	if len(data) >= 4 {
		hasStartCode := (data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x00 && data[3] == 0x01) ||
			(data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01)
		if !hasStartCode {
			logrus.Warnf("Video sample does not have valid H.264 start code: %02x %02x %02x %02x",
				data[0], data[1], data[2], data[3])
		}
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

	// Parse H.264 NAL units from the data
	nalUnits, err := m.parseH264NALUnits(data)
	if err != nil {
		logrus.Errorf("Failed to parse H.264 NAL units: %v", err)
		return
	}

	if len(nalUnits) == 0 {
		logrus.Warnf("No NAL units found in video sample (size: %d bytes). First bytes: %02x %02x %02x %02x",
			len(data),
			safeByte(data, 0), safeByte(data, 1), safeByte(data, 2), safeByte(data, 3))
		return
	}

	logrus.Debugf("Parsed %d NAL units from video sample (total size: %d)", len(nalUnits), len(data))

	// Track if any peers received data
	anyPeerReceived := false

	for _, peer := range m.peers {
		peer.mu.RLock()
		hasVideoTrack := peer.VideoTrack != nil
		isConnected := peer.IsConnected
		peerConnection := peer.Connection
		peer.mu.RUnlock()

		if !hasVideoTrack {
			logrus.Debugf("Peer %s has no video track, skipping", peer.ID)
			continue
		}

		// Check actual connection state - don't rely only on IsConnected flag
		if peerConnection != nil {
			connectionState := peerConnection.ConnectionState()
			iceConnectionState := peerConnection.ICEConnectionState()

			// Only send if connection is actually established
			if connectionState != webrtc.PeerConnectionStateConnected &&
				connectionState != webrtc.PeerConnectionStateConnecting {
				logrus.Debugf("Peer %s connection state is %s, skipping video sample", peer.ID, connectionState.String())
				continue
			}

			// Allow sending during 'checking' to establish connection faster
			// Also allow during 'connected' and 'completed'
			if iceConnectionState != webrtc.ICEConnectionStateConnected &&
				iceConnectionState != webrtc.ICEConnectionStateCompleted &&
				iceConnectionState != webrtc.ICEConnectionStateChecking {
				logrus.Debugf("Peer %s ICE state is %s, skipping video sample", peer.ID, iceConnectionState.String())
				continue
			}
		}

		// Send each NAL unit as a separate sample
		writtenCount := 0

		// Log state for debugging first frame
		logrus.Debugf("Peer %s sending video: PeerConnection=%s, ICE=%s",
			peer.ID,
			peerConnection.ConnectionState().String(),
			peerConnection.ICEConnectionState().String())
		for i, nalUnit := range nalUnits {
			if len(nalUnit) == 0 {
				continue
			}

			// Validate NAL unit (must have at least 1 byte for header)
			if len(nalUnit) < 1 {
				logrus.Warnf("NAL unit %d is too short (size: %d)", i, len(nalUnit))
				continue
			}

			// Log NAL unit type for first few frames
			nalType := nalUnit[0] & 0x1F
			if writtenCount == 0 && i < 5 {
				logrus.Infof("📤 Writing NAL unit %d: type=%d (0x%02x), size=%d, peer=%s, connected=%v",
					i, nalType, nalUnit[0], len(nalUnit), peer.ID, isConnected)
			}

			// Calculate duration based on NAL type
			// I-frames might take longer, P/B frames are quicker
			duration := time.Millisecond * 33 // Default ~30fps
			if nalType == 5 {                 // IDR frame
				duration = time.Millisecond * 100 // IDR frames are keyframes
			}

			sample := media.Sample{
				Data:     nalUnit,
				Duration: duration,
			}
			if timestamp > 0 {
				sample.PacketTimestamp = timestamp
			}

			if err := peer.VideoTrack.WriteSample(sample); err != nil {
				logrus.Errorf("❌ Failed to write video sample to peer %s: %v", peer.ID, err)
			} else {
				writtenCount++
				anyPeerReceived = true
			}
		}

		if writtenCount > 0 {
			logrus.Infof("✅ Wrote %d NAL units to peer %s (total sample size: %d)", writtenCount, peer.ID, len(data))
		} else if len(nalUnits) > 0 {
			logrus.Warnf("⚠️ No NAL units written to peer %s (had %d NAL units)", peer.ID, len(nalUnits))
		}
	}

	if !anyPeerReceived && len(m.peers) > 0 {
		logrus.Warnf("⚠️ No peers received video data (peers: %d, sample size: %d)", len(m.peers), len(data))
		// Log peer states for debugging
		m.peersLock.RLock()
		for id, peer := range m.peers {
			peer.mu.RLock()
			connState := "unknown"
			iceState := "unknown"
			hasTrack := peer.VideoTrack != nil
			if peer.Connection != nil {
				connState = peer.Connection.ConnectionState().String()
				iceState = peer.Connection.ICEConnectionState().String()
			}
			logrus.Warnf("  Peer %s: track=%v, conn=%s, ice=%s", id, hasTrack, connState, iceState)
			peer.mu.RUnlock()
		}
		m.peersLock.RUnlock()
	} else if anyPeerReceived {
		logrus.Debugf("✅ Video sample successfully sent to peers (size: %d)", len(data))
	}
}

// safeByte safely gets a byte at index, returns 0 if out of bounds
func safeByte(data []byte, idx int) byte {
	if idx >= 0 && idx < len(data) {
		return data[idx]
	}
	return 0
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
