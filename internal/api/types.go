package api

// VersionResponse from GET /version
type VersionResponse struct {
	Meta    bool   `json:"meta"`
	Version string `json:"version"`
	Premium bool   `json:"premium,omitempty"`
}

// ConfigResponse from GET /configs
type ConfigResponse struct {
	Port               int    `json:"port"`
	SocksPort          int    `json:"socks-port"`
	RedirPort          int    `json:"redir-port"`
	TProxyPort         int    `json:"tproxy-port"`
	MixedPort          int    `json:"mixed-port"`
	AllowLan           bool   `json:"allow-lan"`
	BindAddress        string `json:"bind-address"`
	Mode               string `json:"mode"`
	LogLevel           string `json:"log-level"`
	IPv6               bool   `json:"ipv6"`
	Tun                TunConfig `json:"tun"`
}

// TunConfig represents TUN interface configuration.
type TunConfig struct {
	Enable bool   `json:"enable"`
	Device string `json:"device,omitempty"`
	Stack  string `json:"stack,omitempty"`
}

// ConfigPatch is used with PATCH /configs to update specific fields.
type ConfigPatch map[string]interface{}

// ProxiesResponse from GET /proxies
type ProxiesResponse struct {
	Proxies map[string]Proxy `json:"proxies"`
}

// Proxy represents a single proxy node or group.
type Proxy struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	UDP     bool     `json:"udp"`
	XUDP    bool     `json:"xudp"`
	History []Delay  `json:"history"`
	All     []string `json:"all,omitempty"`  // Group members (for group types)
	Now     string   `json:"now,omitempty"`  // Currently selected (for Selector)
	Hidden  bool     `json:"hidden,omitempty"`
	Icon    string   `json:"icon,omitempty"`
}

// Delay represents a latency test result.
type Delay struct {
	Time  string `json:"time"`
	Delay int    `json:"delay"` // milliseconds, 0 means timeout
}

// ProxyGroupsResponse from GET /group
type ProxyGroupsResponse struct {
	Proxies map[string]Proxy `json:"proxies"`
}

// DelayTestResponse from GET /proxies/{name}/delay or /group/{name}/delay
type DelayTestResponse struct {
	Delay int `json:"delay,omitempty"`
	// For group delay test, returns map of proxy name -> delay
}

// GroupDelayResponse is the response from GET /group/{name}/delay
type GroupDelayResponse map[string]int

// SelectProxyRequest is the body for PUT /proxies/{name}
type SelectProxyRequest struct {
	Name string `json:"name"`
}

// Connection represents a single active connection.
type Connection struct {
	ID          string         `json:"id"`
	Metadata    ConnMetadata   `json:"metadata"`
	Upload      int64          `json:"upload"`
	Download    int64          `json:"download"`
	Start       string         `json:"start"`
	Chains      []string       `json:"chains"`
	Rule        string         `json:"rule"`
	RulePayload string         `json:"rulePayload"`
}

// ConnMetadata holds connection metadata.
type ConnMetadata struct {
	Network     string `json:"network"`
	Type        string `json:"type"`
	SourceIP    string `json:"sourceIP"`
	SourcePort  string `json:"sourcePort"`
	Destination string `json:"destination,omitempty"`
	Host        string `json:"host"`
	DNSMode     string `json:"dnsMode"`
	Process     string `json:"process,omitempty"`
	ProcessPath string `json:"processPath,omitempty"`
}

// ConnectionsResponse from GET /connections
type ConnectionsResponse struct {
	DownloadTotal int64        `json:"downloadTotal"`
	UploadTotal   int64        `json:"uploadTotal"`
	Connections   []Connection `json:"connections"`
	Memory        int64        `json:"memory,omitempty"`
}

// Rule represents a routing rule.
type Rule struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
	Size    int    `json:"size,omitempty"`
}

// RulesResponse from GET /rules
type RulesResponse struct {
	Rules []Rule `json:"rules"`
}

// ProvidersResponse from GET /providers/proxies
type ProvidersResponse struct {
	Providers map[string]Provider `json:"providers"`
}

// Provider represents a proxy or rule provider.
type Provider struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	VehicleType string  `json:"vehicleType"`
	Proxies     []Proxy `json:"proxies,omitempty"`
	UpdatedAt   string  `json:"updatedAt,omitempty"`
}

// TrafficData from WS /traffic
type TrafficData struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// MemoryData from WS /memory
type MemoryData struct {
	InUse int64 `json:"inuse"`
	OSUse int64 `json:"oslimit,omitempty"`
}

// LogData from WS /logs
type LogData struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}
