package config

import (
	"encoding/json"
	"time"
)

// ═══════════════════════════════════════════════════════════
// Root Config Structure
// ═══════════════════════════════════════════════════════════

// ServerConfig تنظیمات کامل سرور
type ServerConfig struct {
	Version  int            `json:"version"`
	Server   ServerInfo     `json:"server"`
	SNI      SNIConfig      `json:"sni,omitempty"`
	Cover    CoverConfig    `json:"cover_traffic,omitempty"`
	DNS      DNSConfig      `json:"dns_fallback,omitempty"`
	UTLS     UTLSConfig     `json:"utls,omitempty"`
	Fragment FragmentConfig `json:"fragmentation,omitempty"`
	Modes    ModesConfig    `json:"modes,omitempty"`
	Metadata Metadata       `json:"metadata,omitempty"`
}

// ═══════════════════════════════════════════════════════════
// Server Info
// ═══════════════════════════════════════════════════════════

// ServerInfo اطلاعات اصلی سرور
type ServerInfo struct {
	Name     string `json:"name"`
	Address  string `json:"address"`           // "1.2.3.4:8443"
	Protocol string `json:"protocol"`          // "guarch", "grouk", "zhip"
	PSK      string `json:"psk"`               // hex-encoded
	CertPin  string `json:"cert_pin,omitempty"` // sha256 pin
}

// ═══════════════════════════════════════════════════════════
// SNI Config
// ═══════════════════════════════════════════════════════════

// SNIConfig تنظیمات SNI (Server Name Indication)
type SNIConfig struct {
	Enabled             bool        `json:"enabled"`
	Mode                string      `json:"mode"` // "random", "weighted", "sequential", "single"
	Domains             []SNIDomain `json:"domains"`
	RotationInterval    Duration    `json:"rotation_interval,omitempty"`    // مثلاً "5m"
	HealthCheckInterval Duration    `json:"health_check_interval,omitempty"` // مثلاً "30s"
	HealthCheckTimeout  Duration    `json:"health_check_timeout,omitempty"`  // مثلاً "5s"
}

// SNIDomain یک دامنه SNI
type SNIDomain struct {
	Domain      string `json:"domain"`                 // "google.com"
	Weight      int    `json:"weight,omitempty"`       // برای weighted mode
	CheckHealth bool   `json:"check_health,omitempty"` // آیا health check بشه؟
	Fallback    bool   `json:"fallback,omitempty"`     // آیا fallback است؟
	Priority    int    `json:"priority,omitempty"`     // اولویت (پایین‌تر = بالاتر)
}

// ═══════════════════════════════════════════════════════════
// Cover Traffic Config
// ═══════════════════════════════════════════════════════════

// CoverConfig تنظیمات cover traffic
type CoverConfig struct {
	Enabled  bool            `json:"enabled"`
	Mode     string          `json:"mode"` // "auto", "stealth", "balanced", "fast", "off"
	Domains  []CoverDomain   `json:"domains"`
	Adaptive AdaptiveConfig  `json:"adaptive,omitempty"`
}

// CoverDomain یک دامنه برای cover traffic
type CoverDomain struct {
	Domain      string   `json:"domain"`       // "www.google.com"
	Paths       []string `json:"paths"`        // ["/", "/search?q=weather"]
	Weight      int      `json:"weight"`       // وزن برای انتخاب تصادفی
	IntervalMin Duration `json:"interval_min"` // حداقل فاصله بین درخواست‌ها
	IntervalMax Duration `json:"interval_max"` // حداکثر فاصله بین درخواست‌ها
	UserAgents  []string `json:"user_agents,omitempty"` // لیست User-Agent (optional)
}

// AdaptiveConfig تنظیمات adaptive mode
type AdaptiveConfig struct {
	Enabled         bool   `json:"enabled"`
	BatteryAware    bool   `json:"battery_aware"`    // کاهش در حالت کم باتری
	DataSaverMode   bool   `json:"data_saver_mode"`  // کاهش برای صرفه‌جویی دیتا
	IdleThreshold   string `json:"idle_threshold"`   // مثلاً "50KB/min"
	LightThreshold  string `json:"light_threshold"`  // مثلاً "500KB/min"
	MediumThreshold string `json:"medium_threshold"` // مثلاً "5MB/min"
}

// ═══════════════════════════════════════════════════════════
// DNS Fallback Config
// ═══════════════════════════════════════════════════════════

// DNSConfig تنظیمات DNS tunneling fallback
type DNSConfig struct {
	Enabled         bool     `json:"enabled"`
	Domain          string   `json:"domain"`           // "tunnel.example.com"
	Servers         []string `json:"servers"`          // ["8.8.8.8:53", "1.1.1.1:53"]
	AutoSwitch      bool     `json:"auto_switch"`      // خودکار switch به DNS اگه TLS fail شد
	SwitchThreshold int      `json:"switch_threshold"` // بعد از چند fail
	Timeout         Duration `json:"timeout,omitempty"`
}

// ═══════════════════════════════════════════════════════════
// uTLS Config
// ═══════════════════════════════════════════════════════════

// UTLSConfig تنظیمات uTLS (Browser Fingerprinting)
type UTLSConfig struct {
	Enabled     bool     `json:"enabled"`
	Fingerprint string   `json:"fingerprint"` // "chrome_auto", "firefox_121", ...
	Options     []string `json:"options,omitempty"` // لیست fingerprint های قابل انتخاب
	RandomizeALPN bool   `json:"randomize_alpn,omitempty"`
}

// ═══════════════════════════════════════════════════════════
// Fragmentation Config
// ═══════════════════════════════════════════════════════════

// FragmentConfig تنظیمات packet fragmentation
type FragmentConfig struct {
	Enabled bool `json:"enabled"`
	MinSize int  `json:"min_size"` // حداقل سایز fragment (bytes)
	MaxSize int  `json:"max_size"` // حداکثر سایز fragment (bytes)
	Delay   Duration `json:"delay,omitempty"` // تاخیر بین fragment ها
}

// ═══════════════════════════════════════════════════════════
// Modes Config
// ═══════════════════════════════════════════════════════════

// ModesConfig تنظیمات پیش‌فرض برای هر mode
type ModesConfig struct {
	Stealth  ModeSettings `json:"stealth"`
	Balanced ModeSettings `json:"balanced"`
	Fast     ModeSettings `json:"fast"`
}

// ModeSettings تنظیمات یک mode
type ModeSettings struct {
	CoverRate   string `json:"cover_rate"`   // "high", "medium", "low", "off"
	Padding     int    `json:"padding"`      // حداکثر padding (bytes)
	SNIRotation string `json:"sni_rotation"` // "fast", "normal", "slow", "off"
	DNSFallback bool   `json:"dns_fallback,omitempty"` // فعال کردن DNS fallback
}

// ═══════════════════════════════════════════════════════════
// Metadata
// ═══════════════════════════════════════════════════════════

// Metadata اطلاعات متا
type Metadata struct {
	CreatedAt string            `json:"created_at,omitempty"`
	UpdatedAt string            `json:"updated_at,omitempty"`
	Country   string            `json:"country,omitempty"`   // "IR", "US", ...
	ISPHint   string            `json:"isp_hint,omitempty"`  // "MCI", "Irancell", ...
	Notes     string            `json:"notes,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Custom    map[string]string `json:"custom,omitempty"` // custom fields
}

// ═══════════════════════════════════════════════════════════
// Helper Types
// ═══════════════════════════════════════════════════════════

// Duration wrapper برای parse کردن duration strings
type Duration struct {
	time.Duration
}

// UnmarshalJSON parse duration از JSON string
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

// MarshalJSON تبدیل duration به JSON string
func (d Duration) MarshalJSON() ([]byte, error) {
	if d.Duration == 0 {
		return json.Marshal("")
	}
	return json.Marshal(d.Duration.String())
}

// ═══════════════════════════════════════════════════════════
// Client Local Config (برای اپلیکیشن موبایل)
// ═══════════════════════════════════════════════════════════

// ClientConfig تنظیمات محلی کلاینت
type ClientConfig struct {
	Version     int                    `json:"version"`
	Preferences UserPreferences        `json:"user_preferences"`
	Servers     []SavedServer          `json:"servers"`
	ActiveID    string                 `json:"active_server_id,omitempty"`
}

// UserPreferences تنظیمات کاربر
type UserPreferences struct {
	AutoConnect       bool   `json:"auto_connect"`
	AutoSelectFastest bool   `json:"auto_select_fastest"`
	KillSwitch        bool   `json:"kill_switch"`
	IPv6              bool   `json:"ipv6"`
	LogLevel          string `json:"log_level"` // "debug", "info", "warn", "error"
	Theme             string `json:"theme"`     // "dark", "light", "auto"
}

// SavedServer یک سرور ذخیره شده
type SavedServer struct {
	ID             string                 `json:"id"` // UUID
	ImportedConfig ServerConfig           `json:"imported_config"`
	Overrides      *ConfigOverrides       `json:"overrides,omitempty"`
	Stats          ServerStats            `json:"stats,omitempty"`
}

// ConfigOverrides تنظیمات override محلی
type ConfigOverrides struct {
	Mode              *string         `json:"mode,omitempty"` // "stealth", "balanced", "fast"
	CoverEnabled      *bool           `json:"cover_enabled,omitempty"`
	BatteryAware      *bool           `json:"battery_aware,omitempty"`
	DataSaverMode     *bool           `json:"data_saver_mode,omitempty"`
	CustomSNIDomains  []SNIDomain     `json:"custom_sni_domains,omitempty"`
	CustomCoverDomains []CoverDomain  `json:"custom_cover_domains,omitempty"`
}

// ServerStats آمار سرور
type ServerStats struct {
	LastConnected string `json:"last_connected,omitempty"`
	TotalBytes    uint64 `json:"total_bytes"`
	AvgPing       int    `json:"avg_ping"` // milliseconds
	SuccessRate   float64 `json:"success_rate"` // 0.0 - 1.0
	TotalConnections int `json:"total_connections"`
}

// ═══════════════════════════════════════════════════════════
// Helper Functions
// ═══════════════════════════════════════════════════════════

// GetMaxPaddingForMode برگرداندن حداکثر padding بر اساس mode
func GetMaxPaddingForMode(mode string) int {
	switch mode {
	case "stealth":
		return 1024
	case "balanced":
		return 256
	case "fast", "off":
		return 0
	default:
		return 256 // default = balanced
	}
}
