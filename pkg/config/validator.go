package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Validate(cfg *ServerConfig) error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version: %d (expected 1)", cfg.Version)
	}
	
	if err := v.validateServerInfo(&cfg.Server); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	
	if cfg.SNI.Enabled {
		if err := v.validateSNI(&cfg.SNI); err != nil {
			return fmt.Errorf("sni: %w", err)
		}
	}
	
	if cfg.Cover.Enabled {
		if err := v.validateCover(&cfg.Cover); err != nil {
			return fmt.Errorf("cover: %w", err)
		}
	}
	
	if cfg.DNS.Enabled {
		if err := v.validateDNS(&cfg.DNS); err != nil {
			return fmt.Errorf("dns: %w", err)
		}
	}
	
	if cfg.UTLS.Enabled {
		if err := v.validateUTLS(&cfg.UTLS); err != nil {
			return fmt.Errorf("utls: %w", err)
		}
	}
	
	if cfg.Fragment.Enabled {
		if err := v.validateFragment(&cfg.Fragment); err != nil {
			return fmt.Errorf("fragment: %w", err)
		}
	}

	if err := v.validateMetadata(&cfg.Metadata); err != nil {
		return fmt.Errorf("metadata: %w", err)
	}
	
	return nil
}

func (v *Validator) validateServerInfo(info *ServerInfo) error {
	if info.Name == "" {
		return fmt.Errorf("name is required")
	}
	
	if info.Address == "" {
		return fmt.Errorf("address is required")
	}
	
	host, port, err := net.SplitHostPort(info.Address)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}
	
	if host == "" {
		return fmt.Errorf("host is empty")
	}
	
	if port == "" {
		return fmt.Errorf("port is empty")
	}
	
	validProtocols := map[string]bool{
		"guarch": true,
		"grouk":  true,
		"zhip":   true,
	}
	
	if !validProtocols[info.Protocol] {
		return fmt.Errorf("invalid protocol: %s (must be guarch/grouk/zhip)", info.Protocol)
	}
	
	if info.PSK == "" {
		return fmt.Errorf("PSK is required")
	}
	
	if len(info.PSK) < 16 {
		return fmt.Errorf("PSK too short (minimum 16 characters)")
	}
	
	return nil
}

func (v *Validator) validateSNI(sni *SNIConfig) error {
	if len(sni.Domains) == 0 {
		return fmt.Errorf("no domains specified")
	}
	
	validModes := map[string]bool{
		"random":     true,
		"weighted":   true,
		"sequential": true,
		"single":     true,
	}
	
	if !validModes[sni.Mode] {
		return fmt.Errorf("invalid mode: %s", sni.Mode)
	}
	
	totalWeight := 0
	hasFallback := false
	hasHealthCheck := false
	
	for i, d := range sni.Domains {
		if d.Domain == "" {
			return fmt.Errorf("domain %d: empty domain", i)
		}
		
		if !v.isValidDomain(d.Domain) {
			return fmt.Errorf("domain %d: invalid domain format: %s", i, d.Domain)
		}
		
		if d.Weight < 0 {
			return fmt.Errorf("domain %d: negative weight", i)
		}
		
		totalWeight += d.Weight
		
		if d.Fallback {
			hasFallback = true
		}
		if d.CheckHealth {
			hasHealthCheck = true
		}
	}
	
	if sni.Mode == "weighted" && totalWeight == 0 {
		return fmt.Errorf("weighted mode requires at least one domain with weight > 0")
	}
	
	if hasHealthCheck && !hasFallback {
		return fmt.Errorf("health check enabled but no fallback domain defined (at least one domain should have fallback=true)")
	}
	
	return nil
}

func (v *Validator) validateCover(cover *CoverConfig) error {
	if len(cover.Domains) == 0 {
		return fmt.Errorf("no domains specified")
	}
	
	validModes := map[string]bool{
		"auto":     true,
		"stealth":  true,
		"balanced": true,
		"fast":     true,
		"off":      true,
	}
	
	if !validModes[cover.Mode] {
		return fmt.Errorf("invalid mode: %s", cover.Mode)
	}
	
	totalWeight := 0
	
	for i, d := range cover.Domains {
		if d.Domain == "" {
			return fmt.Errorf("domain %d: empty domain", i)
		}
		
		if !v.isValidDomain(d.Domain) {
			return fmt.Errorf("domain %d: invalid domain: %s", i, d.Domain)
		}
		
		if len(d.Paths) == 0 {
			return fmt.Errorf("domain %d: no paths specified", i)
		}
		
		if d.Weight < 0 {
			return fmt.Errorf("domain %d: negative weight", i)
		}
		
		totalWeight += d.Weight
		
		if d.IntervalMin.Duration < 0 {
			return fmt.Errorf("domain %d: negative interval_min", i)
		}
		
		if d.IntervalMax.Duration < 0 {
			return fmt.Errorf("domain %d: negative interval_max", i)
		}
		
		if d.IntervalMax.Duration > 0 && d.IntervalMin.Duration > d.IntervalMax.Duration {
			return fmt.Errorf("domain %d: interval_min > interval_max", i)
		}
	}
	
	if totalWeight == 0 {
		return fmt.Errorf("total weight is zero")
	}
	
	return nil
}

func (v *Validator) validateDNS(dns *DNSConfig) error {
	if dns.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	
	if !v.isValidDomain(dns.Domain) {
		return fmt.Errorf("invalid domain: %s", dns.Domain)
	}
	
	if len(dns.Servers) == 0 {
		return fmt.Errorf("no DNS servers specified")
	}
	
	for i, srv := range dns.Servers {
		if _, _, err := net.SplitHostPort(srv); err != nil {
			return fmt.Errorf("server %d: invalid format: %w", i, err)
		}
	}
	
	if dns.SwitchThreshold < 0 {
		return fmt.Errorf("negative switch threshold")
	}
	
	return nil
}

func (v *Validator) validateUTLS(utls *UTLSConfig) error {
	if utls.Fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}
	
	return nil
}

func (v *Validator) validateFragment(frag *FragmentConfig) error {
	if frag.MinSize < 0 {
		return fmt.Errorf("negative min_size")
	}
	
	if frag.MaxSize < 0 {
		return fmt.Errorf("negative max_size")
	}
	
	if frag.MaxSize > 0 && frag.MinSize > frag.MaxSize {
		return fmt.Errorf("min_size > max_size")
	}
	
	if frag.MinSize < 64 {
		return fmt.Errorf("min_size too small (minimum 64 bytes)")
	}
	
	if frag.MaxSize > 8192 {
		return fmt.Errorf("max_size too large (maximum 8192 bytes)")
	}
	
	return nil
}

func (v *Validator) isValidDomain(domain string) bool {
	domain = strings.TrimPrefix(domain, "www.")
	
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	
	for _, ch := range domain {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '.' || ch == '-') {
			return false
		}
	}
	
	u, err := url.Parse("https://" + domain)
	if err != nil {
		return false
	}
	
	if u.Host != domain {
		return false
	}
	
	return true
}

func (v *Validator) validateMetadata(meta *Metadata) error {
	if meta.Quota != nil && !meta.Quota.Unlimited {
		if meta.Quota.TotalBytes < 0 {
			return fmt.Errorf("quota.total_bytes cannot be negative")
		}
		if meta.Quota.UsedBytes < 0 {
			return fmt.Errorf("quota.used_bytes cannot be negative")
		}
		if meta.Quota.UsedBytes > meta.Quota.TotalBytes {
			return fmt.Errorf("quota.used_bytes exceeds total_bytes")
		}
	}
	
	if meta.Announcement != nil && meta.Announcement.Enabled {
		if meta.Announcement.URL != "" {
			if _, err := url.Parse(meta.Announcement.URL); err != nil {
				return fmt.Errorf("announcement.url invalid: %w", err)
			}
		}
		
		if meta.Announcement.URL == "" && meta.Announcement.Text == "" {
			return fmt.Errorf("announcement requires either url or text")
		}
		
		if meta.Announcement.Interval.Duration < 0 {
			return fmt.Errorf("announcement.interval cannot be negative")
		}
		
		validPriorities := map[string]bool{"info": true, "warning": true, "critical": true}
		if meta.Announcement.Priority != "" && !validPriorities[meta.Announcement.Priority] {
			return fmt.Errorf("announcement.priority must be info/warning/critical")
		}
	}
	
	if meta.ExpiresAt != "" {
		expiry, err := time.Parse(time.RFC3339, meta.ExpiresAt)
		if err != nil {
			return fmt.Errorf("expires_at invalid format (expected RFC3339): %w", err)
		}
		
		if time.Now().After(expiry) {
			return fmt.Errorf("config has expired!")
		}
	}
	
	return nil
}
