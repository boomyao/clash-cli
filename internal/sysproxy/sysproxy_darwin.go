//go:build darwin

package sysproxy

import (
	"fmt"
	"os/exec"
	"strings"
)

func newPlatformProxy() SystemProxy {
	return NewDarwinProxy()
}

// DarwinProxy implements SystemProxy for macOS using networksetup.
type DarwinProxy struct {
	interfaces []string
}

// NewDarwinProxy creates a macOS system proxy manager.
func NewDarwinProxy() *DarwinProxy {
	return &DarwinProxy{
		interfaces: getActiveInterfaces(),
	}
}

// Enable sets HTTP and SOCKS5 proxy on all active network interfaces.
func (d *DarwinProxy) Enable(httpAddr, socksAddr string) error {
	if len(d.interfaces) == 0 {
		d.interfaces = getActiveInterfaces()
	}

	for _, iface := range d.interfaces {
		if httpAddr != "" {
			host, port := splitAddr(httpAddr)
			if err := networksetup("-setwebproxy", iface, host, port); err != nil {
				return fmt.Errorf("set web proxy on %s: %w", iface, err)
			}
			if err := networksetup("-setsecurewebproxy", iface, host, port); err != nil {
				return fmt.Errorf("set secure web proxy on %s: %w", iface, err)
			}
			_ = networksetup("-setwebproxystate", iface, "on")
			_ = networksetup("-setsecurewebproxystate", iface, "on")
		}

		if socksAddr != "" {
			host, port := splitAddr(socksAddr)
			if err := networksetup("-setsocksfirewallproxy", iface, host, port); err != nil {
				return fmt.Errorf("set socks proxy on %s: %w", iface, err)
			}
			_ = networksetup("-setsocksfirewallproxystate", iface, "on")
		}
	}

	return nil
}

// Disable removes proxy settings from all active network interfaces.
func (d *DarwinProxy) Disable() error {
	for _, iface := range d.interfaces {
		_ = networksetup("-setwebproxystate", iface, "off")
		_ = networksetup("-setsecurewebproxystate", iface, "off")
		_ = networksetup("-setsocksfirewallproxystate", iface, "off")
	}
	return nil
}

// IsEnabled checks if any network interface has a proxy enabled.
func (d *DarwinProxy) IsEnabled() (bool, error) {
	for _, iface := range d.interfaces {
		out, err := exec.Command("networksetup", "-getwebproxy", iface).Output()
		if err != nil {
			continue
		}
		if strings.Contains(string(out), "Enabled: Yes") {
			return true, nil
		}
	}
	return false, nil
}

func getActiveInterfaces() []string {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return []string{"Wi-Fi"}
	}

	var interfaces []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "An asterisk") {
			continue
		}
		if strings.Contains(line, "Wi-Fi") ||
			strings.Contains(line, "Ethernet") ||
			strings.Contains(line, "USB") ||
			strings.Contains(line, "Thunderbolt") {
			interfaces = append(interfaces, line)
		}
	}

	if len(interfaces) == 0 {
		interfaces = []string{"Wi-Fi"}
	}
	return interfaces
}

func networksetup(args ...string) error {
	return exec.Command("networksetup", args...).Run()
}

func splitAddr(addr string) (string, string) {
	parts := strings.SplitN(addr, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return addr, "7890"
}
