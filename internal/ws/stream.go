package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Stream manages a WebSocket connection with automatic reconnection.
type Stream struct {
	url    string
	secret string

	mu   sync.Mutex
	conn *websocket.Conn

	stopCh chan struct{}
	msgCh  chan json.RawMessage
}

// NewStream creates a new WebSocket stream.
func NewStream(wsURL, secret string) *Stream {
	return &Stream{
		url:    wsURL,
		secret: secret,
		stopCh: make(chan struct{}),
		msgCh:  make(chan json.RawMessage, 64),
	}
}

// Connect establishes the WebSocket connection and starts reading.
func (s *Stream) Connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	header := http.Header{}
	if s.secret != "" {
		header.Set("Authorization", "Bearer "+s.secret)
	}

	conn, _, err := websocket.DefaultDialer.Dial(s.url, header)
	if err != nil {
		return fmt.Errorf("websocket dial %s: %w", s.url, err)
	}

	s.conn = conn

	// Start reading in background
	go s.readLoop()

	return nil
}

// readLoop continuously reads messages and sends them to the channel.
func (s *Stream) readLoop() {
	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		s.mu.Lock()
		conn := s.conn
		s.mu.Unlock()

		if conn == nil {
			// Try to reconnect
			time.Sleep(2 * time.Second)
			if err := s.reconnect(); err != nil {
				continue
			}
			s.mu.Lock()
			conn = s.conn
			s.mu.Unlock()
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			// Connection lost, set conn to nil to trigger reconnect
			s.mu.Lock()
			if s.conn != nil {
				s.conn.Close()
				s.conn = nil
			}
			s.mu.Unlock()
			continue
		}

		select {
		case s.msgCh <- json.RawMessage(message):
		default:
			// Drop message if channel is full (backpressure)
		}
	}
}

// reconnect tries to re-establish the WebSocket connection.
func (s *Stream) reconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		s.conn.Close()
	}

	header := http.Header{}
	if s.secret != "" {
		header.Set("Authorization", "Bearer "+s.secret)
	}

	conn, _, err := websocket.DefaultDialer.Dial(s.url, header)
	if err != nil {
		return err
	}

	s.conn = conn
	return nil
}

// Messages returns the channel of raw JSON messages.
func (s *Stream) Messages() <-chan json.RawMessage {
	return s.msgCh
}

// Close shuts down the WebSocket connection.
func (s *Stream) Close() {
	close(s.stopCh)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
}
