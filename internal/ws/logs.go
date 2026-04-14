package ws

import (
	"encoding/json"
	"fmt"

	"github.com/boomyao/clash-cli/internal/api"
	tea "github.com/charmbracelet/bubbletea"
)

// LogMsg carries a log entry to bubbletea.
type LogMsg struct {
	Type    string
	Payload string
	Err     error
}

// LogStream wraps a WebSocket stream for /logs.
type LogStream struct {
	stream *Stream
}

// NewLogStream creates a log stream with the specified level filter.
func NewLogStream(baseWSURL, secret, level string) *LogStream {
	url := fmt.Sprintf("%s/logs?level=%s", baseWSURL, level)
	return &LogStream{
		stream: NewStream(url, secret),
	}
}

// Connect starts the WebSocket connection.
func (l *LogStream) Connect() error {
	return l.stream.Connect()
}

// Close stops the stream.
func (l *LogStream) Close() {
	l.stream.Close()
}

// WaitForLog returns a tea.Cmd that waits for the next log entry.
func (l *LogStream) WaitForLog() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-l.stream.Messages()
		if !ok {
			return LogMsg{Err: fmt.Errorf("log stream closed")}
		}
		var data api.LogData
		if err := json.Unmarshal(msg, &data); err != nil {
			return LogMsg{Err: err}
		}
		return LogMsg{Type: data.Type, Payload: data.Payload}
	}
}
