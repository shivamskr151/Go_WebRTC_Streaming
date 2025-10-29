package rtmp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	webrtcmanager "golang-webrtc-streaming/internal/webrtc"

	"github.com/deepch/vdk/format/flv"
	"github.com/sirupsen/logrus"
)

type Server struct {
	port          int
	webrtcManager *webrtcmanager.Manager
	listener      net.Listener
	isRunning     bool
	mu            sync.RWMutex
	clients       map[string]*Client
	clientsLock   sync.RWMutex
}

type Client struct {
	conn          net.Conn
	webrtcManager *webrtcmanager.Manager
	isActive      bool
	mu            sync.RWMutex
}

func NewServer(port int, webrtcManager *webrtcmanager.Manager) *Server {
	return &Server{
		port:          port,
		webrtcManager: webrtcManager,
		clients:       make(map[string]*Client),
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("RTMP server is already running")
	}

	// Start listening
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start RTMP server: %w", err)
	}

	s.listener = listener
	s.isRunning = true

	logrus.Infof("RTMP server started on port %d", s.port)

	// Start accepting connections in goroutine
	go s.acceptConnections(ctx)

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}

	// Close all clients
	s.clientsLock.Lock()
	for _, client := range s.clients {
		client.Close()
	}
	s.clients = make(map[string]*Client)
	s.clientsLock.Unlock()

	s.isRunning = false
	logrus.Info("RTMP server stopped")
	return nil
}

func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

func (s *Server) acceptConnections(ctx context.Context) {
	defer func() {
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			logrus.Info("RTMP server context cancelled")
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if s.isRunning {
					logrus.Errorf("Failed to accept RTMP connection: %v", err)
				}
				continue
			}

			// Handle connection in goroutine
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	clientID := fmt.Sprintf("rtmp_%d", time.Now().UnixNano())
	logrus.Infof("New RTMP connection: %s", clientID)

	client := &Client{
		conn:          conn,
		webrtcManager: s.webrtcManager,
		isActive:      true,
	}

	s.clientsLock.Lock()
	s.clients[clientID] = client
	s.clientsLock.Unlock()

	defer func() {
		s.clientsLock.Lock()
		delete(s.clients, clientID)
		s.clientsLock.Unlock()
		logrus.Infof("RTMP client disconnected: %s", clientID)
	}()

	// Handle RTMP handshake and streaming
	if err := s.handleRTMPStream(client); err != nil {
		logrus.Errorf("RTMP stream error for client %s: %v", clientID, err)
	}
}

func (s *Server) handleRTMPStream(client *Client) error {
	// RTMP handshake
	if err := s.performHandshake(client.conn); err != nil {
		return fmt.Errorf("RTMP handshake failed: %w", err)
	}

	// Create FLV demuxer
	demuxer := flv.NewDemuxer(client.conn)

	// Get codec data
	codecData, err := demuxer.Streams()
	if err != nil {
		return fmt.Errorf("failed to get stream codec data: %w", err)
	}

	logrus.Infof("RTMP stream codec data: %+v", codecData)

	// Process packets
	for {
		client.mu.RLock()
		if !client.isActive {
			client.mu.RUnlock()
			break
		}
		client.mu.RUnlock()

		pkt, err := demuxer.ReadPacket()
		if err != nil {
			return fmt.Errorf("failed to read RTMP packet: %w", err)
		}

		// Convert packet to WebRTC sample
		if pkt.IsKeyFrame {
			timestamp := uint32(pkt.Time.Nanoseconds() / 1000000) // Convert to milliseconds
			s.webrtcManager.WriteVideoSample(pkt.Data, timestamp)
		}
	}

	return nil
}

func (s *Server) performHandshake(conn net.Conn) error {
	// Simplified RTMP handshake
	// In production, you'd want a more complete implementation

	// Read C0 + C1
	c0c1 := make([]byte, 1537)
	if _, err := conn.Read(c0c1); err != nil {
		return err
	}

	// Send S0 + S1 + S2
	s0s1s2 := make([]byte, 3073)
	s0s1s2[0] = 0x03 // RTMP version

	// Copy C1 to S2
	copy(s0s1s2[1537:], c0c1[1:])

	if _, err := conn.Write(s0s1s2); err != nil {
		return err
	}

	// Read C2
	c2 := make([]byte, 1536)
	if _, err := conn.Read(c2); err != nil {
		return err
	}

	return nil
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isActive = false
	if c.conn != nil {
		c.conn.Close()
	}
}

func (s *Server) GetClientCount() int {
	s.clientsLock.RLock()
	defer s.clientsLock.RUnlock()
	return len(s.clients)
}
