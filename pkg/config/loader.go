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
		if IsLegacyClientFormat(data) {
			legacyCfg, err := ParseLegacyClient(data)
			if err != nil {
				return nil, fmt.Errorf("config: parse legacy client: %w", err)
			}
			cfg = *legacyCfg
		} else if IsLegacyServerFormat(data) {
			legacyCfg, err := ParseLegacyServer(data)
			if err != nil {
				return nil, fmt.Errorf("config: parse legacy server: %w", err)
			}
			cfg = *legacyCfg
		} else {
			return nil, fmt.Errorf("config: parse JSON: %w", err)
		}
	}
	l.applyMinimalDefaults(&cfg)
	if err := l.validator.Validate(&cfg); err != nil {
		return nil, fmt.Errorf("config: validation: %w", err)
	}
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
	var jsonData []byte
	var err error
	jsonData, err = base64.StdEncoding.DecodeString(payload)
	if err != nil {
		jsonData, err = base64.URLEncoding.DecodeString(payload)
		if err != nil {
			jsonData, err = base64.RawURLEncoding.DecodeString(payload)
			if err != nil {
				jsonData, err = base64.RawStdEncoding.DecodeString(payload)
				if err != nil {
					return nil, fmt.Errorf("config: decode base64: %w", err)
				}
			}
		}
	}
	var rawCfg ServerConfig
	if err := json.Unmarshal(jsonData, &rawCfg); err != nil {
		return nil, fmt.Errorf("config: parse URI payload: %w", err)
	}
	if rawCfg.Server.Protocol == "" {
		rawCfg.Server.Protocol = protocol
	}
	l.applyMinimalDefaults(&rawCfg)
	if err := l.validator.Validate(&rawCfg); err != nil {
		return nil, fmt.Errorf("config: validation: %w", err)
	}
	return &rawCfg, nil
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

func (l *Loader) applyMinimalDefaults(cfg *ServerConfig) {
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.SocksPort == 0 {
		cfg.SocksPort = 7070
	}
	if cfg.Transport != nil {
		if cfg.Transport.Type == "" {
			cfg.Transport.Type = "direct"
		}
		if cfg.Transport.Port == 0 {
			cfg.Transport.Port = 443
		}
	}
	if cfg.Server.Protocol == "grouk" && cfg.Grouk != nil {
		if cfg.Grouk.FECGroupSize == 0 {
			cfg.Grouk.FECGroupSize = 4
		}
		if cfg.Grouk.FECGroupSize < 2 {
			cfg.Grouk.FECGroupSize = 2
		}
		if cfg.Grouk.FECGroupSize > 16 {
			cfg.Grouk.FECGroupSize = 16
		}
	}
	if cfg.Modes != nil {
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
	}
	if cfg.Metadata != nil {
		if cfg.Metadata.CreatedAt == "" {
			cfg.Metadata.CreatedAt = time.Now().Format(time.RFC3339)
		}
		cfg.Metadata.UpdatedAt = time.Now().Format(time.RFC3339)
		if cfg.Metadata.Announcement != nil && cfg.Metadata.Announcement.Enabled {
			if cfg.Metadata.Announcement.Interval.Duration == 0 {
				cfg.Metadata.Announcement.Interval.Duration = 1 * time.Hour
			}
			if cfg.Metadata.Announcement.Priority == "" {
				cfg.Metadata.Announcement.Priority = "info"
			}
		}
		if cfg.Metadata.Quota != nil && !cfg.Metadata.Quota.Unlimited {
			if cfg.Metadata.Quota.TotalBytes > 0 && cfg.Metadata.Quota.RemainingBytes == 0 && cfg.Metadata.Quota.UsedBytes > 0 {
				remaining := cfg.Metadata.Quota.TotalBytes - cfg.Metadata.Quota.UsedBytes
				if remaining < 0 {
					remaining = 0
				}
				cfg.Metadata.Quota.RemainingBytes = remaining
			}
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
