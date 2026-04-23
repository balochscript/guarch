// pkg/config/presets.go
package config

import (
	"time"
)

// Presets تنظیمات از پیش آماده
var Presets = map[string]*ServerConfig{
	"iran_stealth":   IranStealthPreset(),
	"iran_balanced":  IranBalancedPreset(),
	"global_stealth": GlobalStealthPreset(),
	"global_balanced": GlobalBalancedPreset(),
	"minimal":        MinimalPreset(),
}

// GetPreset دریافت preset با نام
func GetPreset(name string) (*ServerConfig, bool) {
	preset, ok := Presets[name]
	if !ok {
		return nil, false
	}
	
	// Clone کردن تا original تغییر نکنه
	loader := NewLoader()
	cloned, err := loader.Clone(preset)
	if err != nil {
		return nil, false
	}
	
	return cloned, true
}

// ListPresets لیست همه preset ها
func ListPresets() []string {
	names := make([]string, 0, len(Presets))
	for name := range Presets {
		names = append(names, name)
	}
	return names
}

// ═══════════════════════════════════════════════════════════
// Iran Presets (بهینه شده برای ایران)
// ═══════════════════════════════════════════════════════════

// IranStealthPreset حداکثر مخفی‌کاری برای ایران
func IranStealthPreset() *ServerConfig {
	return &ServerConfig{
		Version: 1,
		Server: ServerInfo{
			Name:     "Iran Stealth Server",
			Address:  "YOUR_SERVER:8443",
			Protocol: "guarch",
			PSK:      "REPLACE_WITH_YOUR_PSK",
			CertPin:  "",
		},
		
		SNI: SNIConfig{
			Enabled: true,
			Mode:    "weighted",
			Domains: []SNIDomain{
				// سایت‌های محبوب ایرانی
				{Domain: "digikala.com", Weight: 25, CheckHealth: true},
				{Domain: "aparat.com", Weight: 20, CheckHealth: true},
				{Domain: "snapp.ir", Weight: 15, CheckHealth: true},
				{Domain: "divar.ir", Weight: 15, CheckHealth: true},
				
				// Fallback به سایت‌های بین‌المللی
				{Domain: "cloudflare.com", Weight: 10, Fallback: true},
				{Domain: "microsoft.com", Weight: 10, Fallback: true},
				{Domain: "apple.com", Weight: 5, Fallback: true},
			},
			RotationInterval:    Duration{3 * time.Minute},
			HealthCheckInterval: Duration{30 * time.Second},
			HealthCheckTimeout:  Duration{5 * time.Second},
		},
		
		Cover: CoverConfig{
			Enabled: true,
			Mode:    "stealth",
			Domains: []CoverDomain{
				{
					Domain: "www.google.com",
					Paths:  []string{"/", "/search?q=weather", "/search?q=news", "/maps"},
					Weight: 30,
					IntervalMin: Duration{2 * time.Second},
					IntervalMax: Duration{8 * time.Second},
				},
				{
					Domain: "www.microsoft.com",
					Paths:  []string{"/", "/windows", "/office", "/azure"},
					Weight: 20,
					IntervalMin: Duration{3 * time.Second},
					IntervalMax: Duration{10 * time.Second},
				},
				{
					Domain: "github.com",
					Paths:  []string{"/", "/explore", "/trending"},
					Weight: 15,
					IntervalMin: Duration{4 * time.Second},
					IntervalMax: Duration{12 * time.Second},
				},
				{
					Domain: "stackoverflow.com",
					Paths:  []string{"/", "/questions", "/tags"},
					Weight: 15,
					IntervalMin: Duration{3 * time.Second},
					IntervalMax: Duration{10 * time.Second},
				},
				{
					Domain: "www.cloudflare.com",
					Paths:  []string{"/", "/learning", "/products"},
					Weight: 10,
					IntervalMin: Duration{5 * time.Second},
					IntervalMax: Duration{15 * time.Second},
				},
				{
					Domain: "learn.microsoft.com",
					Paths:  []string{"/", "/docs", "/training"},
					Weight: 10,
					IntervalMin: Duration{4 * time.Second},
					IntervalMax: Duration{12 * time.Second},
				},
			},
			Adaptive: AdaptiveConfig{
				Enabled:         true,
				BatteryAware:    true,
				DataSaverMode:   false,
				IdleThreshold:   "50KB/min",
				LightThreshold:  "500KB/min",
				MediumThreshold: "5MB/min",
			},
		},
		
		DNS: DNSConfig{
			Enabled:         true,
			Domain:          "tunnel.example.com",
			Servers:         []string{"8.8.8.8:53", "1.1.1.1:53", "208.67.222.222:53"},
			AutoSwitch:      true,
			SwitchThreshold: 3,
			Timeout:         Duration{5 * time.Second},
		},
		
		UTLS: UTLSConfig{
			Enabled:     true,
			Fingerprint: "chrome_auto",
			Options:     []string{"chrome_120", "chrome_119", "firefox_121", "edge_120"},
		},
		
		Fragment: FragmentConfig{
			Enabled: false,
			MinSize: 64,
			MaxSize: 256,
		},
		
		Modes: ModesConfig{
			Stealth: ModeSettings{
				CoverRate:   "high",
				Padding:     1024,
				SNIRotation: "fast",
				DNSFallback: true,
			},
			Balanced: ModeSettings{
				CoverRate:   "medium",
				Padding:     256,
				SNIRotation: "normal",
				DNSFallback: true,
			},
			Fast: ModeSettings{
				CoverRate:   "off",
				Padding:     0,
				SNIRotation: "off",
				DNSFallback: false,
			},
		},
		
		Metadata: Metadata{
			Country: "IR",
			ISPHint: "MCI/Irancell",
			Notes:   "Optimized for Iranian networks with heavy filtering",
			Tags:    []string{"iran", "stealth", "mci", "irancell"},
		},
	}
}

// IranBalancedPreset تعادل بین سرعت و امنیت برای ایران
func IranBalancedPreset() *ServerConfig {
	cfg := IranStealthPreset()
	
	// تغییرات برای balanced mode
	cfg.Server.Name = "Iran Balanced Server"
	cfg.Cover.Mode = "balanced"
	
	// کاهش cover traffic
	for i := range cfg.Cover.Domains {
		cfg.Cover.Domains[i].IntervalMin.Duration *= 2
		cfg.Cover.Domains[i].IntervalMax.Duration *= 2
	}
	
	// کاهش SNI rotation
	cfg.SNI.RotationInterval.Duration = 5 * time.Minute
	
	// فعال کردن data saver
	cfg.Cover.Adaptive.DataSaverMode = true
	
	cfg.Metadata.Notes = "Balanced mode for Iranian networks - good speed and stealth"
	cfg.Metadata.Tags = []string{"iran", "balanced", "recommended"}
	
	return cfg
}

// ═══════════════════════════════════════════════════════════
// Global Presets (بین‌المللی)
// ═══════════════════════════════════════════════════════════

// GlobalStealthPreset حداکثر مخفی‌کاری (بین‌المللی)
func GlobalStealthPreset() *ServerConfig {
	return &ServerConfig{
		Version: 1,
		Server: ServerInfo{
			Name:     "Global Stealth Server",
			Address:  "YOUR_SERVER:8443",
			Protocol: "guarch",
			PSK:      "REPLACE_WITH_YOUR_PSK",
			CertPin:  "",
		},
		
		SNI: SNIConfig{
			Enabled: true,
			Mode:    "weighted",
			Domains: []SNIDomain{
				{Domain: "google.com", Weight: 30, CheckHealth: true},
				{Domain: "microsoft.com", Weight: 25, CheckHealth: true},
				{Domain: "cloudflare.com", Weight: 20, CheckHealth: true},
				{Domain: "apple.com", Weight: 15, Fallback: true},
				{Domain: "github.com", Weight: 10, Fallback: true},
			},
			RotationInterval:    Duration{5 * time.Minute},
			HealthCheckInterval: Duration{1 * time.Minute},
			HealthCheckTimeout:  Duration{5 * time.Second},
		},
		
		Cover: CoverConfig{
			Enabled: true,
			Mode:    "stealth",
			Domains: []CoverDomain{
				{
					Domain: "www.google.com",
					Paths:  []string{"/", "/search?q=technology", "/news"},
					Weight: 35,
					IntervalMin: Duration{2 * time.Second},
					IntervalMax: Duration{10 * time.Second},
				},
				{
					Domain: "www.microsoft.com",
					Paths:  []string{"/", "/windows", "/microsoft-365"},
					Weight: 25,
					IntervalMin: Duration{3 * time.Second},
					IntervalMax: Duration{12 * time.Second},
				},
				{
					Domain: "github.com",
					Paths:  []string{"/", "/explore", "/trending"},
					Weight: 20,
					IntervalMin: Duration{4 * time.Second},
					IntervalMax: Duration{15 * time.Second},
				},
				{
					Domain: "stackoverflow.com",
					Paths:  []string{"/", "/questions"},
					Weight: 20,
					IntervalMin: Duration{3 * time.Second},
					IntervalMax: Duration{10 * time.Second},
				},
			},
			Adaptive: AdaptiveConfig{
				Enabled:         true,
				BatteryAware:    true,
				DataSaverMode:   false,
				IdleThreshold:   "100KB/min",
				LightThreshold:  "1MB/min",
				MediumThreshold: "10MB/min",
			},
		},
		
		DNS: DNSConfig{
			Enabled:         false, // معمولاً در شبکه‌های بین‌المللی نیاز نیست
			Domain:          "tunnel.example.com",
			Servers:         []string{"8.8.8.8:53", "1.1.1.1:53"},
			AutoSwitch:      false,
			SwitchThreshold: 5,
		},
		
		UTLS: UTLSConfig{
			Enabled:     true,
			Fingerprint: "chrome_auto",
			Options:     []string{"chrome_120", "firefox_121", "safari_17"},
		},
		
		Fragment: FragmentConfig{
			Enabled: false,
		},
		
		Modes: ModesConfig{
			Stealth: ModeSettings{
				CoverRate:   "high",
				Padding:     1024,
				SNIRotation: "fast",
			},
			Balanced: ModeSettings{
				CoverRate:   "medium",
				Padding:     256,
				SNIRotation: "normal",
			},
			Fast: ModeSettings{
				CoverRate:   "off",
				Padding:     0,
				SNIRotation: "off",
			},
		},
		
		Metadata: Metadata{
			Country: "Global",
			Notes:   "General purpose stealth configuration",
			Tags:    []string{"global", "stealth"},
		},
	}
}

// GlobalBalancedPreset تعادل برای شبکه‌های بین‌المللی
func GlobalBalancedPreset() *ServerConfig {
	cfg := GlobalStealthPreset()
	
	cfg.Server.Name = "Global Balanced Server"
	cfg.Cover.Mode = "balanced"
	
	// کاهش cover traffic
	for i := range cfg.Cover.Domains {
		cfg.Cover.Domains[i].Weight /= 2
		if cfg.Cover.Domains[i].Weight < 5 {
			cfg.Cover.Domains[i].Weight = 5
		}
	}
	
	cfg.SNI.RotationInterval.Duration = 10 * time.Minute
	
	cfg.Metadata.Notes = "Balanced configuration for global use"
	cfg.Metadata.Tags = []string{"global", "balanced", "recommended"}
	
	return cfg
}

// ═══════════════════════════════════════════════════════════
// Minimal Preset (حداقل overhead)
// ═══════════════════════════════════════════════════════════

// MinimalPreset حداقل تنظیمات (سرعت بالا)
func MinimalPreset() *ServerConfig {
	return &ServerConfig{
		Version: 1,
		Server: ServerInfo{
			Name:     "Minimal Server",
			Address:  "YOUR_SERVER:8443",
			Protocol: "guarch",
			PSK:      "REPLACE_WITH_YOUR_PSK",
		},
		
		SNI: SNIConfig{
			Enabled: false,
		},
		
		Cover: CoverConfig{
			Enabled: false,
		},
		
		DNS: DNSConfig{
			Enabled: false,
		},
		
		UTLS: UTLSConfig{
			Enabled: false,
		},
		
		Fragment: FragmentConfig{
			Enabled: false,
		},
		
		Modes: ModesConfig{
			Fast: ModeSettings{
				CoverRate:   "off",
				Padding:     0,
				SNIRotation: "off",
			},
		},
		
		Metadata: Metadata{
			Notes: "Minimal configuration - maximum speed, no stealth features",
			Tags:  []string{"minimal", "fast", "no-censorship"},
		},
	}
}

// ═══════════════════════════════════════════════════════════
// Helper Functions
// ═══════════════════════════════════════════════════════════

// CreateIranConfig ساخت config برای ایران با تنظیمات دلخواه
func CreateIranConfig(serverAddr, psk string, mode string) *ServerConfig {
	var cfg *ServerConfig
	
	switch mode {
	case "stealth":
		cfg = IranStealthPreset()
	case "balanced":
		cfg = IranBalancedPreset()
	default:
		cfg = IranBalancedPreset()
	}
	
	cfg.Server.Address = serverAddr
	cfg.Server.PSK = psk
	
	return cfg
}

// CreateGlobalConfig ساخت config بین‌المللی
func CreateGlobalConfig(serverAddr, psk string, mode string) *ServerConfig {
	var cfg *ServerConfig
	
	switch mode {
	case "stealth":
		cfg = GlobalStealthPreset()
	case "balanced":
		cfg = GlobalBalancedPreset()
	default:
		cfg = GlobalBalancedPreset()
	}
	
	cfg.Server.Address = serverAddr
	cfg.Server.PSK = psk
	
	return cfg
}
