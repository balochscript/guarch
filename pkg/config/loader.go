package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Loader struct {
	validator *Validator
}

func NewLoader() *Loader {
	return &Loader{
		validator: NewValidator(),
	}
}

func (l *Loader) LoadFromJSON(data []byte) (*ServerConfig, error) {
	var cfg ServerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse JSON: %w", err)
	}
	
	if err := l.validator.Validate(&cfg); err != nil {
		return nil, fmt.Errorf("config: validation: %w", err)
	}
	
	l.applyDefaults(&cfg)
	
	return &cfg, nil
}

func (l *Loader) LoadFromFile(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read file: %w", err)
	}
	return l.LoadFromJSON(data)
}

func (l *Loader) LoadFromURI(uri string) (*ServerConfig, error) {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("config: invalid URI format (expected protocol://payload)")
	}
	
	protocol := parts[0]
	payload := parts[1]
	
	jsonData, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		jsonData, err = base64.URLEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("config: decode base64: %w", err)
		}
	}
	
	cfg, err := l.LoadFromJSON(jsonData)
	if err != nil {
		return nil, err
	}
	
	if cfg.Server.Protocol == "" {
		cfg.Server.Protocol = protocol
	}
	
	return cfg, nil
}

func (l *Loader) ExportToURI(cfg *ServerConfig) (string, error) {
	if err := l.validator.Validate(cfg); err != nil {
		return "", fmt.Errorf("config: validation failed: %w", err)
	}
	
	jsonData, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("config: marshal JSON: %w", err)
	}
	
	b64 := base64.URLEncoding.EncodeToString(jsonData)
	
	protocol := cfg.Server.Protocol
	if protocol == "" {
		protocol = "guarch"
	}
	
	return fmt.Sprintf("%s://%s", protocol, b64), nil
}

func (l *Loader) ExportToJSON(cfg *ServerConfig, indent bool) ([]byte, error) {
	if err := l.validator.Validate(cfg); err != nil {
		return nil, fmt.Errorf("config: validation failed: %w", err)
	}
	
	if indent {
		return json.MarshalIndent(cfg, "", "  ")
	}
	return json.Marshal(cfg)
}

func (l *Loader) SaveToFile(cfg *ServerConfig, path string) error {
	data, err := l.ExportToJSON(cfg, true)
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0600)
}

func (l *Loader) applyDefaults(cfg *ServerConfig) {
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	if cfg.SocksPort == 0 {
		cfg.SocksPort = 1080
	}
	
	if cfg.SNI.Enabled {
		if cfg.SNI.Mode == "" {
			cfg.SNI.Mode = "random"
		}
		if cfg.SNI.RotationInterval.Duration == 0 {
			cfg.SNI.RotationInterval.Duration = 5 * time.Minute
		}
		if cfg.SNI.HealthCheckInterval.Duration == 0 {
			cfg.SNI.HealthCheckInterval.Duration = 30 * time.Second
		}
		if cfg.SNI.HealthCheckTimeout.Duration == 0 {
			cfg.SNI.HealthCheckTimeout.Duration = 5 * time.Second
		}
		
		for i := range cfg.SNI.Domains {
			if cfg.SNI.Domains[i].Weight == 0 {
				cfg.SNI.Domains[i].Weight = 10
			}
		}
	}
	
	if cfg.Cover.Enabled {
		if cfg.Cover.Mode == "" {
			cfg.Cover.Mode = "auto"
		}
		
		for i := range cfg.Cover.Domains {
			if cfg.Cover.Domains[i].Weight == 0 {
				cfg.Cover.Domains[i].Weight = 10
			}
			if cfg.Cover.Domains[i].IntervalMin.Duration == 0 {
				cfg.Cover.Domains[i].IntervalMin.Duration = 2 * time.Second
			}
			if cfg.Cover.Domains[i].IntervalMax.Duration == 0 {
				cfg.Cover.Domains[i].IntervalMax.Duration = 10 * time.Second
			}
		}
		
		if cfg.Cover.Adaptive.IdleThreshold == "" {
			cfg.Cover.Adaptive.IdleThreshold = "50KB/min"
		}
		if cfg.Cover.Adaptive.LightThreshold == "" {
			cfg.Cover.Adaptive.LightThreshold = "500KB/min"
		}
		if cfg.Cover.Adaptive.MediumThreshold == "" {
			cfg.Cover.Adaptive.MediumThreshold = "5MB/min"
		}
	}
	
	if cfg.DNS.Enabled {
		if len(cfg.DNS.Servers) == 0 {
			cfg.DNS.Servers = []string{"8.8.8.8:53", "1.1.1.1:53"}
		}
		if cfg.DNS.SwitchThreshold == 0 {
			cfg.DNS.SwitchThreshold = 3
		}
		if cfg.DNS.Timeout.Duration == 0 {
			cfg.DNS.Timeout.Duration = 5 * time.Second
		}
	}
	
	if cfg.UTLS.Enabled && cfg.UTLS.Fingerprint == "" {
		cfg.UTLS.Fingerprint = "chrome_auto"
	}
	
	if cfg.Fragment.Enabled {
		if cfg.Fragment.MinSize == 0 {
			cfg.Fragment.MinSize = 64
		}
		if cfg.Fragment.MaxSize == 0 {
			cfg.Fragment.MaxSize = 256
		}
	}
	
	if cfg.Modes.Stealth.CoverRate == "" {
		cfg.Modes.Stealth.CoverRate = "high"
	}
	if cfg.Modes.Stealth.Padding == 0 {
		cfg.Modes.Stealth.Padding = 1024
	}
	if cfg.Modes.Stealth.SNIRotation == "" {
		cfg.Modes.Stealth.SNIRotation = "fast"
	}
	
	if cfg.Modes.Balanced.CoverRate == "" {
		cfg.Modes.Balanced.CoverRate = "medium"
	}
	if cfg.Modes.Balanced.Padding == 0 {
		cfg.Modes.Balanced.Padding = 256
	}
	if cfg.Modes.Balanced.SNIRotation == "" {
		cfg.Modes.Balanced.SNIRotation = "normal"
	}
	
	if cfg.Modes.Fast.CoverRate == "" {
		cfg.Modes.Fast.CoverRate = "off"
	}
	if cfg.Modes.Fast.SNIRotation == "" {
		cfg.Modes.Fast.SNIRotation = "off"
	}
	
	if cfg.Metadata.CreatedAt == "" {
		cfg.Metadata.CreatedAt = time.Now().Format(time.RFC3339)
	}
	
	if cfg.Metadata.Announcement != nil && cfg.Metadata.Announcement.Enabled {
		if cfg.Metadata.Announcement.Interval.Duration == 0 {
			cfg.Metadata.Announcement.Interval.Duration = 1 * time.Hour
		}
		if cfg.Metadata.Announcement.Priority == "" {
			cfg.Metadata.Announcement.Priority = "info"
		}
	}
	
	if cfg.Metadata.Quota != nil && !cfg.Metadata.Quota.Unlimited {
		if cfg.Metadata.Quota.RemainingBytes == 0 {
			cfg.Metadata.Quota.RemainingBytes = cfg.Metadata.Quota.TotalBytes - cfg.Metadata.Quota.UsedBytes
		}
	}
}

func (l *Loader) Clone(cfg *ServerConfig) (*ServerConfig, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	
	var cloned ServerConfig
	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}
	
	return &cloned, nil
}
