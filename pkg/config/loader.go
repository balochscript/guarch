// pkg/config/loader.go
package config

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "os"
    "strings"
)

type Loader struct {
    validator *Validator
}

func NewLoader() *Loader {
    return &Loader{
        validator: NewValidator(),
    }
}

// LoadFromJSON بارگذاری از JSON bytes
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

// LoadFromFile بارگذاری از فایل
func (l *Loader) LoadFromFile(path string) (*ServerConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("config: read file: %w", err)
    }
    return l.LoadFromJSON(data)
}

// LoadFromURI بارگذاری از URI
// Format: guarch://base64-json
func (l *Loader) LoadFromURI(uri string) (*ServerConfig, error) {
    // Parse protocol
    parts := strings.SplitN(uri, "://", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("config: invalid URI format")
    }
    
    protocol := parts[0]
    payload := parts[1]
    
    // Decode base64
    jsonData, err := base64.StdEncoding.DecodeString(payload)
    if err != nil {
        return nil, fmt.Errorf("config: decode base64: %w", err)
    }
    
    // Load JSON
    cfg, err := l.LoadFromJSON(jsonData)
    if err != nil {
        return nil, err
    }
    
    // Set protocol from URI if not set
    if cfg.Server.Protocol == "" {
        cfg.Server.Protocol = protocol
    }
    
    return cfg, nil
}

// ExportToURI صادر کردن به URI
func (l *Loader) ExportToURI(cfg *ServerConfig) (string, error) {
    // Validate
    if err := l.validator.Validate(cfg); err != nil {
        return "", err
    }
    
    // Marshal to JSON
    jsonData, err := json.Marshal(cfg)
    if err != nil {
        return "", err
    }
    
    // Encode base64
    b64 := base64.StdEncoding.EncodeToString(jsonData)
    
    // Build URI
    protocol := cfg.Server.Protocol
    if protocol == "" {
        protocol = "guarch"
    }
    
    return fmt.Sprintf("%s://%s", protocol, b64), nil
}

// applyDefaults اعمال مقادیر پیش‌فرض
func (l *Loader) applyDefaults(cfg *ServerConfig) {
    if cfg.Version == 0 {
        cfg.Version = 1
    }
    
    if cfg.SNI.Enabled && cfg.SNI.RotationInterval.Duration == 0 {
        cfg.SNI.RotationInterval.Duration = 5 * time.Minute
    }
    
    if cfg.Cover.Enabled && cfg.Cover.Mode == "" {
        cfg.Cover.Mode = "auto"
    }
    
    // ... سایر defaults
}
