//go:build linux

package sysproxy

import (
	"fmt"
	"os/exec"
	"strings"
)

func newPlatformProxy() SystemProxy {
	return NewLinuxProxy()
}

// LinuxProxy implements SystemProxy for Linux using gsettings (GNOME).
type LinuxProxy struct{}

// NewLinuxProxy creates a Linux system proxy manager.
func NewLinuxProxy() *LinuxProxy {
	return &LinuxProxy{}
}

// Enable sets HTTP and SOCKS5 proxy via gsettings.
func (l *LinuxProxy) Enable(httpAddr, socksAddr string) error {
	if httpAddr != "" {
		host, port := splitAddr(httpAddr)
		if err := gsettings("org.gnome.system.proxy.http", "host", host); err != nil {
			return fmt.Errorf("set http proxy host: %w", err)
		}
		if err := gsettings("org.gnome.system.proxy.http", "port", port); err != nil {
			return fmt.Errorf("set http proxy port: %w", err)
		}
		if err := gsettings("org.gnome.system.proxy.https", "host", host); err != nil {
			return fmt.Errorf("set https proxy host: %w", err)
		}
		if err := gsettings("org.gnome.system.proxy.https", "port", port); err != nil {
			return fmt.Errorf("set https proxy port: %w", err)
		}
	}

	if socksAddr != "" {
		host, port := splitAddr(socksAddr)
		if err := gsettings("org.gnome.system.proxy.socks", "host", host); err != nil {
			return fmt.Errorf("set socks proxy host: %w", err)
		}
		if err := gsettings("org.gnome.system.proxy.socks", "port", port); err != nil {
			return fmt.Errorf("set socks proxy port: %w", err)
		}
	}

	if err := exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run(); err != nil {
		return fmt.Errorf("set proxy mode: %w", err)
	}

	return nil
}

// Disable removes proxy settings.
func (l *LinuxProxy) Disable() error {
	return exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
}

// IsEnabled checks if the system proxy is set to manual mode.
func (l *LinuxProxy) IsEnabled() (bool, error) {
	out, err := exec.Command("gsettings", "get", "org.gnome.system.proxy", "mode").Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(out), "manual"), nil
}

func gsettings(schema, key, value string) error {
	return exec.Command("gsettings", "set", schema, key, value).Run()
}

func splitAddr(addr string) (string, string) {
	parts := strings.SplitN(addr, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return addr, "7890"
}
