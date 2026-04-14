package tui

// Shared message types used across multiple pages.

// WindowSizeMsg is sent when the terminal window is resized.
// This is re-broadcast to all pages.
type WindowSizeMsg struct {
	Width  int
	Height int
}

// TrafficMsg carries real-time traffic data from WebSocket.
type TrafficMsg struct {
	Up   int64
	Down int64
}

// MemoryMsg carries real-time memory usage from WebSocket.
type MemoryMsg struct {
	InUse int64
}

// CoreStatusMsg reports whether the mihomo core is running.
type CoreStatusMsg struct {
	Running bool
	Version string
}

// SysProxyStatusMsg reports the system proxy state.
type SysProxyStatusMsg struct {
	Enabled bool
}

// ConfigModeMsg reports the current clash mode.
type ConfigModeMsg struct {
	Mode string // "rule", "global", "direct"
}

// ErrorMsg is a generic error message shown as a toast.
type ErrorMsg struct {
	Err     error
	Context string
}

// InfoMsg is a generic info message shown as a toast.
type InfoMsg struct {
	Message string
}

// PageActivatedMsg is sent to a page when it becomes the active tab.
type PageActivatedMsg struct{}

// PageDeactivatedMsg is sent to a page when it loses focus.
type PageDeactivatedMsg struct{}
