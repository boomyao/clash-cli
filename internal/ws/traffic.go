package ws

import (
	"encoding/json"
	"fmt"

	"github.com/boomyao/clash-cli/internal/api"
	tea "github.com/charmbracelet/bubbletea"
)

// TrafficMsg carries traffic data to bubbletea.
type TrafficMsg struct {
	Up   int64
	Down int64
	Err  error
}

// TrafficStream wraps a WebSocket stream for /traffic.
type TrafficStream struct {
	stream *Stream
}

// NewTrafficStream creates a traffic stream.
func NewTrafficStream(baseWSURL, secret string) *TrafficStream {
	url := fmt.Sprintf("%s/traffic", baseWSURL)
	return &TrafficStream{
		stream: NewStream(url, secret),
	}
}

// Connect starts the WebSocket connection.
func (t *TrafficStream) Connect() error {
	return t.stream.Connect()
}

// Close stops the stream.
func (t *TrafficStream) Close() {
	t.stream.Close()
}

// WaitForTraffic returns a tea.Cmd that waits for the next traffic message.
func (t *TrafficStream) WaitForTraffic() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-t.stream.Messages()
		if !ok {
			return TrafficMsg{Err: fmt.Errorf("traffic stream closed")}
		}
		var data api.TrafficData
		if err := json.Unmarshal(msg, &data); err != nil {
			return TrafficMsg{Err: err}
		}
		return TrafficMsg{Up: data.Up, Down: data.Down}
	}
}
