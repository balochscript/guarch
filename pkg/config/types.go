package config

import (
	"encoding/json"
	"time"
)

type ServerConfig struct {
	Version   int               `json:"version"`
	Server    ServerInfo        `json:"server"`
	Transport *TransportConfig  `json:"transport,omitempty"`
	SocksPort int               `json:"socks_port,omitempty"`
	SNI       *SNIConfig        `json:"sni,omitempty"`
	Cover     *CoverConfig      `json:"cover,omitempty"`
	DNS       *DNSConfig        `json:"dns,omitempty"`
	UTLS      *UTLSConfig       `json:"utls,omitempty"`
	Fragment  *FragmentConfig   `json:"fragmentation,omitempty"`
	Grouk     *GroukConfig      `json:"grouk,omitempty"`
	Zhip      *ZhipConfig       `json:"zhip,omitempty"`
	
	PaddingEnabled          bool   `json:"padding_enabled,omitempty"`
	MaxPadding              int    `json:"max_padding,omitempty"`
	TrafficPattern          string `json:"traffic_pattern,omitempty"`
	BatteryThreshold        int    `json:"battery_threshold,omitempty"`
	HysteresisDelay         int    `json:"hysteresis_delay,omitempty"`
	DecoyEnabled            bool   `json:"decoy_enabled,omitempty"`
	ProbeDetectionEnabled   bool   `json:"probe_detection_enabled,omitempty"`
	ProbeMaxRate            int    `json:"probe_max_rate,omitempty"`
	ProbeWindow             int    `json:"probe_window,omitempty"`
	
	Modes     *ModesConfig      `json:"modes,omitempty"`
	Metadata  *Metadata         `json:"metadata,omitempty"`
	
	PreferIPv6 bool `json:"prefer_ipv6,omitempty"`
}

type ServerInfo struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Protocol string `json:"protocol"`
	PSK      string `json:"psk"`
	CertPin  string `json:"cert_pin,omitempty"`
}

type TransportConfig struct {
	Type             string                      `json:"type"`
	Host             string                      `json:"host,omitempty"`
	Port             int                         `json:"port,omitempty"`
	Path             string                      `json:"path,omitempty"`
	UseTLS           bool                        `json:"use_tls"`
	Headers          map[string]string           `json:"headers,omitempty"`
	DialTimeout      int                         `json:"dial_timeout,omitempty"`
	HandshakeTimeout int                         `json:"handshake_timeout,omitempty"`
	FallbackOrder    []string                    `json:"fallback_order,omitempty"`
	Experimental     ExperimentalTransportConfig `json:"experimental,omitempty"`
}

type ExperimentalTransportConfig struct {
	EnableHTTP2 bool `json:"enable_http2"`
}

type GroukConfig struct {
	EnableFEC       bool `json:"enable_fec"`
	FECDataShards   int  `json:"fec_data_shards"`
	FECParityShards int  `json:"fec_parity_shards"`
	FECGroupSize    int  `json:"fec_group_size"`
}

type ZhipConfig struct {
	MaxIdleTimeout  int `json:"max_idle_timeout,omitempty"`
	KeepAlivePeriod int `json:"keepalive_period,omitempty"`
	MaxStreams      int `json:"max_streams,omitempty"`
}

type SNIConfig struct {
	Enabled             bool        `json:"enabled"`
	Mode                string      `json:"mode"`
	Domains             []SNIDomain `json:"domains"`
	RotationInterval    Duration    `json:"rotation_interval,omitempty"`
	HealthCheckInterval Duration    `json:"health_check_interval,omitempty"`
	HealthCheckTimeout  Duration    `json:"health_check_timeout,omitempty"`
}

type SNIDomain struct {
	Domain      string `json:"domain"`
	Weight      int    `json:"weight,omitempty"`
	CheckHealth bool   `json:"check_health,omitempty"`
	Fallback    bool   `json:"fallback,omitempty"`
	Priority    int    `json:"priority,omitempty"`
}

type CoverConfig struct {
	Enabled  bool           `json:"enabled"`
	Mode     string         `json:"mode"`
	Domains  []CoverDomain  `json:"domains"`
	Adaptive AdaptiveConfig `json:"adaptive,omitempty"`
}

type CoverDomain struct {
	Domain      string   `json:"domain"`
	Paths       []string `json:"paths"`
	Weight      int      `json:"weight"`
	IntervalMin Duration `json:"interval_min"`
	IntervalMax Duration `json:"interval_max"`
	UserAgents  []string `json:"user_agents,omitempty"`
}

type AdaptiveConfig struct {
	Enabled         bool   `json:"enabled"`
	BatteryAware    bool   `json:"battery_aware"`
	DataSaverMode   bool   `json:"data_saver_mode"`
	IdleThreshold   string `json:"idle_threshold"`
	LightThreshold  string `json:"light_threshold"`
	MediumThreshold string `json:"medium_threshold"`
}

type DNSConfig struct {
	Enabled         bool     `json:"enabled"`
	Domain          string   `json:"domain"`
	Servers         []string `json:"servers"`
	AutoSwitch      bool     `json:"auto_switch"`
	SwitchThreshold int      `json:"switch_threshold"`
	Timeout         Duration `json:"timeout,omitempty"`
	MaxRetries      int      `json:"max_retries,omitempty"`
	RetryDelay      Duration `json:"retry_delay,omitempty"`
	ListenAddr      string   `json:"listen_addr,omitempty"`
	MaxSessions     int      `json:"max_sessions,omitempty"`
	SessionTimeout  Duration `json:"session_timeout,omitempty"`
	RateLimit       int      `json:"rate_limit,omitempty"`
	CacheEnabled    bool     `json:"cache_enabled,omitempty"`
	CacheTTL        Duration `json:"cache_ttl,omitempty"`
	BufferSize      int      `json:"buffer_size,omitempty"`
	MaxPacketSize   int      `json:"max_packet_size,omitempty"`
	Compression     bool     `json:"compression,omitempty"`
}

type UTLSConfig struct {
	Enabled       bool     `json:"enabled"`
	Fingerprint   string   `json:"fingerprint"`
	Options       []string `json:"options,omitempty"`
	RandomizeALPN bool     `json:"randomize_alpn,omitempty"`
}

type FragmentConfig struct {
	Enabled bool     `json:"enabled"`
	MinSize int      `json:"min_size"`
	MaxSize int      `json:"max_size"`
	Delay   Duration `json:"delay,omitempty"`
}

type ModesConfig struct {
	Stealth  ModeSettings `json:"stealth"`
	Balanced ModeSettings `json:"balanced"`
	Fast     ModeSettings `json:"fast"`
}

type ModeSettings struct {
	CoverRate   string `json:"cover_rate"`
	Padding     int    `json:"padding"`
	SNIRotation string `json:"sni_rotation"`
	DNSFallback bool   `json:"dns_fallback,omitempty"`
}

type Metadata struct {
	CreatedAt    string              `json:"created_at,omitempty"`
	UpdatedAt    string              `json:"updated_at,omitempty"`
	ExpiresAt    string              `json:"expires_at,omitempty"`
	Country      string              `json:"country,omitempty"`
	ISPHint      string              `json:"isp_hint,omitempty"`
	Notes        string              `json:"notes,omitempty"`
	Tags         []string            `json:"tags,omitempty"`
	Custom       map[string]string   `json:"custom,omitempty"`
	Quota        *QuotaInfo          `json:"quota,omitempty"`
	Announcement *AnnouncementConfig `json:"announcement,omitempty"`
}

type QuotaInfo struct {
	TotalBytes     int64  `json:"total_bytes,omitempty"`
	UsedBytes      int64  `json:"used_bytes,omitempty"`
	RemainingBytes int64  `json:"remaining_bytes,omitempty"`
	ResetDate      string `json:"reset_date,omitempty"`
	Unlimited      bool   `json:"unlimited,omitempty"`
}

type AnnouncementConfig struct {
	Enabled  bool     `json:"enabled"`
	URL      string   `json:"url,omitempty"`
	Text     string   `json:"text,omitempty"`
	Interval Duration `json:"interval,omitempty"`
	Priority string   `json:"priority,omitempty"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if s == "" {
		d.Duration = 0
		return nil
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	d.Duration = dur
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	if d.Duration == 0 {
		return json.Marshal("")
	}
	return json.Marshal(d.Duration.String())
}

type ClientConfig struct {
	Version     int             `json:"version"`
	Preferences UserPreferences `json:"user_preferences"`
	Servers     []SavedServer   `json:"servers"`
	ActiveID    string          `json:"active_server_id,omitempty"`
}

type UserPreferences struct {
	AutoConnect       bool   `json:"auto_connect"`
	AutoSelectFastest bool   `json:"auto_select_fastest"`
	KillSwitch        bool   `json:"kill_switch"`
	IPv6              bool   `json:"ipv6"`
	LogLevel          string `json:"log_level"`
	Theme             string `json:"theme"`
}

type SavedServer struct {
	ID             string           `json:"id"`
	ImportedConfig ServerConfig     `json:"imported_config"`
	Overrides      *ConfigOverrides `json:"overrides,omitempty"`
	Stats          ServerStats      `json:"stats,omitempty"`
}

type ConfigOverrides struct {
	Mode               *string       `json:"mode,omitempty"`
	CoverEnabled       *bool         `json:"cover_enabled,omitempty"`
	BatteryAware       *bool         `json:"battery_aware,omitempty"`
	DataSaverMode      *bool         `json:"data_saver_mode,omitempty"`
	CustomSNIDomains   []SNIDomain   `json:"custom_sni_domains,omitempty"`
	CustomCoverDomains []CoverDomain `json:"custom_cover_domains,omitempty"`
}

type ServerStats struct {
	LastConnected    string  `json:"last_connected,omitempty"`
	TotalBytes       uint64  `json:"total_bytes"`
	AvgPing          int     `json:"avg_ping"`
	SuccessRate      float64 `json:"success_rate"`
	TotalConnections int     `json:"total_connections"`
}

func GetMaxPaddingForMode(mode string) int {
	switch mode {
	case "stealth":
		return 1024
	case "balanced":
		return 256
	case "fast", "off":
		return 0
	default:
		return 256
	}
}
