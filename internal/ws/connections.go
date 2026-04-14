package ws

import (
	"encoding/json"
	"fmt"

	"github.com/boomyao/clash-cli/internal/api"
	tea "github.com/charmbracelet/bubbletea"
)

// ConnectionsMsg carries connections snapshot to bubbletea.
type ConnectionsMsg struct {
	Connections   []api.Connection
	DownloadTotal int64
	UploadTotal   int64
	Memory        int64
	Err           error
}

// ConnectionsStream wraps a WebSocket stream for /connections.
type ConnectionsStream struct {
	stream *Stream
}

// NewConnectionsStream creates a connections stream.
func NewConnectionsStream(baseWSURL, secret string) *ConnectionsStream {
	url := fmt.Sprintf("%s/connections", baseWSURL)
	return &ConnectionsStream{
		stream: NewStream(url, secret),
	}
}

// Connect starts the WebSocket connection.
func (c *ConnectionsStream) Connect() error {
	return c.stream.Connect()
}

// Close stops the stream.
func (c *ConnectionsStream) Close() {
	c.stream.Close()
}

// WaitForConnections returns a tea.Cmd that waits for the next connections snapshot.
func (c *ConnectionsStream) WaitForConnections() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-c.stream.Messages()
		if !ok {
			return ConnectionsMsg{Err: fmt.Errorf("connections stream closed")}
		}
		var data api.ConnectionsResponse
		if err := json.Unmarshal(msg, &data); err != nil {
			return ConnectionsMsg{Err: err}
		}
		return ConnectionsMsg{
			Connections:   data.Connections,
			DownloadTotal: data.DownloadTotal,
			UploadTotal:   data.UploadTotal,
			Memory:        data.Memory,
		}
	}
}
