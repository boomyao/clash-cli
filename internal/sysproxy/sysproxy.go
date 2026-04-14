package sysproxy

// SystemProxy is the interface for managing system-level proxy settings.
type SystemProxy interface {
	// Enable sets the system proxy to the given HTTP/SOCKS addresses.
	Enable(httpAddr, socksAddr string) error
	// Disable removes the system proxy settings.
	Disable() error
	// IsEnabled checks if a system proxy is currently set.
	IsEnabled() (bool, error)
}

// New returns the appropriate SystemProxy implementation for the current platform.
// Returns nil if no implementation is available.
func New() SystemProxy {
	return newPlatformProxy()
}
