package ws

import (
	"encoding/json"
	"fmt"

	"github.com/boomyao/clash-cli/internal/api"
	tea "github.com/charmbracelet/bubbletea"
)

// MemoryMsg carries memory data to bubbletea.
type MemoryMsg struct {
	InUse int64
	Err   error
}

// MemoryStream wraps a WebSocket stream for /memory.
type MemoryStream struct {
	stream *Stream
}

// NewMemoryStream creates a memory stream.
func NewMemoryStream(baseWSURL, secret string) *MemoryStream {
	url := fmt.Sprintf("%s/memory", baseWSURL)
	return &MemoryStream{
		stream: NewStream(url, secret),
	}
}

// Connect starts the WebSocket connection.
func (m *MemoryStream) Connect() error {
	return m.stream.Connect()
}

// Close stops the stream.
func (m *MemoryStream) Close() {
	m.stream.Close()
}

// WaitForMemory returns a tea.Cmd that waits for the next memory message.
func (m *MemoryStream) WaitForMemory() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.stream.Messages()
		if !ok {
			return MemoryMsg{Err: fmt.Errorf("memory stream closed")}
		}
		var data api.MemoryData
		if err := json.Unmarshal(msg, &data); err != nil {
			return MemoryMsg{Err: err}
		}
		return MemoryMsg{InUse: data.InUse}
	}
}
