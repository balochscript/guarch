package config

import "time"

// ═══════════════════════════════════════════════════════════
// Root Config Structure
// ═══════════════════════════════════════════════════════════

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

type ServerInfo struct {
    Name     string `json:"name"`
    Address  string `json:"address"`
    Protocol string `json:"protocol"` // "guarch", "grouk", "zhip"
    PSK      string `json:"psk"`
    CertPin  string `json:"cert_pin,omitempty"`
}

// ═══════════════════════════════════════════════════════════
// SNI Config
// ═══════════════════════════════════════════════════════════

type SNIConfig struct {
    Enabled             bool        `json:"enabled"`
    Mode                string      `json:"mode"` // "random", "weighted", "sequential"
    Domains             []SNIDomain `json:"domains"`
    RotationInterval    Duration    `json:"rotation_interval,omitempty"`
    HealthCheckInterval Duration    `json:"health_check_interval,omitempty"`
}

type SNIDomain struct {
    Domain      string `json:"domain"`
    Weight      int    `json:"weight,omitempty"`
    CheckHealth bool   `json:"check_health,omitempty"`
    Fallback    bool   `json:"fallback,omitempty"`
}

// ═══════════════════════════════════════════════════════════
// Cover Traffic Config
// ═══════════════════════════════════════════════════════════

type CoverConfig struct {
    Enabled  bool            `json:"enabled"`
    Mode     string          `json:"mode"` // "auto", "stealth", "balanced", "fast", "off"
    Domains  []CoverDomain   `json:"domains"`
    Adaptive AdaptiveConfig  `json:"adaptive,omitempty"`
}

type CoverDomain struct {
    Domain      string   `json:"domain"`
    Paths       []string `json:"paths"`
    Weight      int      `json:"weight"`
    IntervalMin Duration `json:"interval_min"`
    IntervalMax Duration `json:"interval_max"`
}

type AdaptiveConfig struct {
    Enabled         bool   `json:"enabled"`
    BatteryAware    bool   `json:"battery_aware"`
    DataSaverMode   bool   `json:"data_saver_mode"`
    IdleThreshold   string `json:"idle_threshold"`   // e.g., "50KB/min"
    LightThreshold  string `json:"light_threshold"`  // e.g., "500KB/min"
    MediumThreshold string `json:"medium_threshold"` // e.g., "5MB/min"
}

// ═══════════════════════════════════════════════════════════
// DNS Fallback Config
// ═══════════════════════════════════════════════════════════

type DNSConfig struct {
    Enabled         bool     `json:"enabled"`
    Domain          string   `json:"domain"`
    Servers         []string `json:"servers"`
    AutoSwitch      bool     `json:"auto_switch"`
    SwitchThreshold int      `json:"switch_threshold"`
}

// ═══════════════════════════════════════════════════════════
// uTLS Config
// ═══════════════════════════════════════════════════════════

type UTLSConfig struct {
    Enabled     bool     `json:"enabled"`
    Fingerprint string   `json:"fingerprint"` // "chrome_auto", "firefox_auto", ...
    Options     []string `json:"options,omitempty"`
}

// ═══════════════════════════════════════════════════════════
// Fragmentation Config
// ═══════════════════════════════════════════════════════════

type FragmentConfig struct {
    Enabled bool `json:"enabled"`
    MinSize int  `json:"min_size"`
    MaxSize int  `json:"max_size"`
}

// ═══════════════════════════════════════════════════════════
// Modes Config
// ═══════════════════════════════════════════════════════════

type ModesConfig struct {
    Stealth  ModeSettings `json:"stealth"`
    Balanced ModeSettings `json:"balanced"`
    Fast     ModeSettings `json:"fast"`
}

type ModeSettings struct {
    CoverRate   string `json:"cover_rate"`   // "high", "medium", "low", "off"
    Padding     int    `json:"padding"`
    SNIRotation string `json:"sni_rotation"` // "fast", "normal", "slow", "off"
}

// ═══════════════════════════════════════════════════════════
// Metadata
// ═══════════════════════════════════════════════════════════

type Metadata struct {
    CreatedAt string `json:"created_at,omitempty"`
    Country   string `json:"country,omitempty"`
    ISPHint   string `json:"isp_hint,omitempty"`
    Notes     string `json:"notes,omitempty"`
}

// ═══════════════════════════════════════════════════════════
// Helper Types
// ═══════════════════════════════════════════════════════════

// Duration برای parse کردن duration strings مثل "5m", "30s"
type Duration struct {
    time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
    var s string
    if err := json.Unmarshal(b, &s); err != nil {
        return err
    }
    dur, err := time.ParseDuration(s)
    if err != nil {
        return err
    }
    d.Duration = dur
    return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
    return json.Marshal(d.Duration.String())
}
