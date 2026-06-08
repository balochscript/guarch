package config

import (
	"time"
)

func GetPreset(name string) (*ServerConfig, bool) {
	var base *ServerConfig
	
	switch name {
	case "iran_stealth":
		base = IranStealthPreset()
	case "iran_balanced":
		base = IranBalancedPreset()
	case "iran_whitelist":
		base = IranWhitelistPreset()
	case "global_stealth":
		base = GlobalStealthPreset()
	case "global_balanced":
		base = GlobalBalancedPreset()
	case "minimal":
		base = MinimalPreset()
	case "iran_grouk":
		base = IranGroukPreset()
	case "iran_grouk_balanced":
		base = IranGroukBalanced()
	case "global_grouk":
		base = GlobalGroukPreset()
	default:
		return nil, false
	}
	
	loader := NewLoader()
	cloned, err := loader.Clone(base)
	if err != nil {
		return nil, false
	}
	
	return cloned, true
}

func ListPresets() []string {
	return []string{
		"iran_stealth",
		"iran_balanced",
		"iran_whitelist",
		"global_stealth",
		"global_balanced",
		"minimal",
		"iran_grouk",
		"iran_grouk_balanced",
		"global_grouk",
	}
}

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
		
		SNI: &SNIConfig{
			Enabled: true,
			Mode:    "weighted",
			Domains: []SNIDomain{
				{Domain: "digikala.com", Weight: 25, CheckHealth: true},
				{Domain: "aparat.com", Weight: 20, CheckHealth: true},
				{Domain: "snapp.ir", Weight: 15, CheckHealth: true},
				{Domain: "divar.ir", Weight: 15, CheckHealth: true},
				{Domain: "cloudflare.com", Weight: 10, Fallback: true},
				{Domain: "microsoft.com", Weight: 10, Fallback: true},
				{Domain: "apple.com", Weight: 5, Fallback: true},
			},
			RotationInterval:    Duration{3 * time.Minute},
			HealthCheckInterval: Duration{30 * time.Second},
			HealthCheckTimeout:  Duration{5 * time.Second},
		},
		
		Cover: &CoverConfig{
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
		
		DNS: &DNSConfig{
			Enabled:         true,
			Domain:          "tunnel.example.com",
			Servers:         []string{"8.8.8.8:53", "1.1.1.1:53", "208.67.222.222:53"},
			AutoSwitch:      true,
			SwitchThreshold: 3,
			Timeout:         Duration{5 * time.Second},
		},
		
		UTLS: &UTLSConfig{
			Enabled:     true,
			Fingerprint: "chrome_auto",
			Options:     []string{"chrome_120", "chrome_119", "firefox_121", "edge_120"},
		},
		
		Fragment: &FragmentConfig{
			Enabled: false,
			MinSize: 64,
			MaxSize: 256,
		},
		
		Modes: &ModesConfig{
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
		
		Metadata: &Metadata{
			Country: "IR",
			ISPHint: "MCI/Irancell",
			Notes:   "Optimized for Iranian networks with heavy filtering",
			Tags:    []string{"iran", "stealth", "mci", "irancell"},
		},
	}
}

func IranBalancedPreset() *ServerConfig {
	cfg := IranStealthPreset()
	
	cfg.Server.Name = "Iran Balanced Server"
	cfg.Cover.Mode = "balanced"
	
	for i := range cfg.Cover.Domains {
		cfg.Cover.Domains[i].IntervalMin.Duration *= 2
		cfg.Cover.Domains[i].IntervalMax.Duration *= 2
	}
	
	cfg.SNI.RotationInterval.Duration = 5 * time.Minute
	
	cfg.Cover.Adaptive.DataSaverMode = true
	
	cfg.Metadata.Notes = "Balanced mode for Iranian networks - good speed and stealth"
	cfg.Metadata.Tags = []string{"iran", "balanced", "recommended"}
	
	return cfg
}

func IranWhitelistPreset() *ServerConfig {
	return &ServerConfig{
		Version: 1,
		Server: ServerInfo{
			Name:     "Iran Whitelist Bypass",
			Address:  "YOUR_SERVER:8443",
			Protocol: "guarch",
			PSK:      "REPLACE_WITH_YOUR_PSK",
			CertPin:  "",
		},
		
		Transport: &TransportConfig{
			Type:   "websocket",
			Host:   "digikala.com",
			Port:   443,
			Path:   "/api/v1/tunnel",
			UseTLS: true,
			Headers: map[string]string{
				"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			},
			FallbackOrder:    []string{"http2", "dns"},
			DialTimeout:      30,
			HandshakeTimeout: 15,
		},
		
		SNI: &SNIConfig{
			Enabled: true,
			Mode:    "weighted",
			Domains: []SNIDomain{
				{Domain: "digikala.com", Weight: 30, CheckHealth: true},
				{Domain: "aparat.com", Weight: 25, CheckHealth: true},
				{Domain: "snapp.ir", Weight: 20, CheckHealth: true},
				{Domain: "divar.ir", Weight: 15, CheckHealth: true},
				{Domain: "shaparak.ir", Weight: 10, Fallback: true},
			},
			RotationInterval:    Duration{3 * time.Minute},
			HealthCheckInterval: Duration{30 * time.Second},
			HealthCheckTimeout:  Duration{5 * time.Second},
		},
		
		Cover: &CoverConfig{
			Enabled: true,
			Mode:    "stealth",
			Domains: []CoverDomain{
				{
					Domain: "digikala.com",
					Paths:  []string{"/", "/search", "/product-list", "/incredible-offers"},
					Weight: 35,
					IntervalMin: Duration{2 * time.Second},
					IntervalMax: Duration{8 * time.Second},
				},
				{
					Domain: "aparat.com",
					Paths:  []string{"/", "/video", "/categories", "/live"},
					Weight: 30,
					IntervalMin: Duration{3 * time.Second},
					IntervalMax: Duration{10 * time.Second},
				},
				{
					Domain: "snapp.ir",
					Paths:  []string{"/", "/services", "/about"},
					Weight: 20,
					IntervalMin: Duration{4 * time.Second},
					IntervalMax: Duration{12 * time.Second},
				},
				{
					Domain: "divar.ir",
					Paths:  []string{"/", "/s/tehran", "/s/mashhad"},
					Weight: 15,
					IntervalMin: Duration{3 * time.Second},
					IntervalMax: Duration{10 * time.Second},
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
		
		DNS: &DNSConfig{
			Enabled:         true,
			Domain:          "tunnel.example.com",
			Servers:         []string{"8.8.8.8:53", "1.1.1.1:53", "208.67.222.222:53"},
			AutoSwitch:      true,
			SwitchThreshold: 3,
			Timeout:         Duration{5 * time.Second},
		},
		
		UTLS: &UTLSConfig{
			Enabled:     true,
			Fingerprint: "chrome_auto",
			Options:     []string{"chrome_120", "edge_120", "firefox_121"},
		},
		
		Fragment: &FragmentConfig{
			Enabled: false,
		},
		
		Modes: &ModesConfig{
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
				CoverRate:   "low",
				Padding:     64,
				SNIRotation: "slow",
				DNSFallback: false,
			},
		},
		
		Metadata: &Metadata{
			Country: "IR",
			ISPHint: "MCI/Irancell/TCI",
			Notes:   "Whitelist bypass using Iranian allowed domains (digikala, aparat, snapp, divar)",
			Tags:    []string{"iran", "whitelist", "websocket", "stealth", "recommended"},
		},
	}
}

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
		
		SNI: &SNIConfig{
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
		
		Cover: &CoverConfig{
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
		
		DNS: &DNSConfig{
			Enabled:         false,
			Domain:          "tunnel.example.com",
			Servers:         []string{"8.8.8.8:53", "1.1.1.1:53"},
			AutoSwitch:      false,
			SwitchThreshold: 5,
		},
		
		UTLS: &UTLSConfig{
			Enabled:     true,
			Fingerprint: "chrome_auto",
			Options:     []string{"chrome_120", "firefox_121", "safari_17"},
		},
		
		Fragment: &FragmentConfig{
			Enabled: false,
		},
		
		Modes: &ModesConfig{
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
		
		Metadata: &Metadata{
			Country: "Global",
			Notes:   "General purpose stealth configuration",
			Tags:    []string{"global", "stealth"},
		},
	}
}

func GlobalBalancedPreset() *ServerConfig {
	cfg := GlobalStealthPreset()
	
	cfg.Server.Name = "Global Balanced Server"
	cfg.Cover.Mode = "balanced"
	
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

func MinimalPreset() *ServerConfig {
	return &ServerConfig{
		Version: 1,
		Server: ServerInfo{
			Name:     "Minimal Server",
			Address:  "YOUR_SERVER:8443",
			Protocol: "guarch",
			PSK:      "REPLACE_WITH_YOUR_PSK",
		},
		
		SNI: &SNIConfig{
			Enabled: false,
		},
		
		Cover: &CoverConfig{
			Enabled: false,
		},
		
		DNS: &DNSConfig{
			Enabled: false,
		},
		
		UTLS: &UTLSConfig{
			Enabled: false,
		},
		
		Fragment: &FragmentConfig{
			Enabled: false,
		},
		
		Modes: &ModesConfig{
			Fast: ModeSettings{
				CoverRate:   "off",
				Padding:     0,
				SNIRotation: "off",
			},
		},
		
		Metadata: &Metadata{
			Notes: "Minimal configuration - maximum speed, no stealth features",
			Tags:  []string{"minimal", "fast", "no-censorship"},
		},
	}
}

func IranGroukPreset() *ServerConfig {
	return &ServerConfig{
		Version: 1,
		Server: ServerInfo{
			Name:     "Iran Grouk Server (FEC)",
			Address:  "YOUR_SERVER:8443",
			Protocol: "grouk",
			PSK:      "REPLACE_WITH_YOUR_PSK",
		},
		Grouk: &GroukConfig{
			EnableFEC:    true,
			FECGroupSize: 4,
		},
		DNS: &DNSConfig{
			Enabled: false,
		},
		Metadata: &Metadata{
			Country: "IR",
			ISPHint: "MCI/Irancell",
			Notes:   "UDP-based Grouk with FEC for lossy networks",
			Tags:    []string{"iran", "grouk", "udp", "fec"},
		},
	}
}

func IranGroukBalanced() *ServerConfig {
	cfg := IranGroukPreset()
	cfg.Server.Name = "Iran Grouk Balanced"
	cfg.Grouk.FECGroupSize = 6
	cfg.Metadata.Notes = "Balanced Grouk with moderate FEC"
	cfg.Metadata.Tags = []string{"iran", "grouk", "balanced", "fec"}
	return cfg
}

func GlobalGroukPreset() *ServerConfig {
	return &ServerConfig{
		Version: 1,
		Server: ServerInfo{
			Name:     "Global Grouk Server",
			Address:  "YOUR_SERVER:8443",
			Protocol: "grouk",
			PSK:      "REPLACE_WITH_YOUR_PSK",
		},
		Grouk: &GroukConfig{
			EnableFEC:    false,
			FECGroupSize: 4,
		},
		DNS: &DNSConfig{
			Enabled: false,
		},
		Metadata: &Metadata{
			Country: "Global",
			Notes:   "UDP-based Grouk for low-latency",
			Tags:    []string{"global", "grouk", "udp", "fast"},
		},
	}
}

func CreateIranConfig(serverAddr, psk string, mode string) *ServerConfig {
	var cfg *ServerConfig
	
	switch mode {
	case "stealth":
		cfg = IranStealthPreset()
	case "balanced":
		cfg = IranBalancedPreset()
	case "whitelist":
		cfg = IranWhitelistPreset()
	default:
		cfg = IranBalancedPreset()
	}
	
	cfg.Server.Address = serverAddr
	cfg.Server.PSK = psk
	
	return cfg
}

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
