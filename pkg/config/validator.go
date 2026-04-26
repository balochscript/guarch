package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Validator اعتبارسنجی config
type Validator struct{}

// NewValidator ساخت validator جدید
func NewValidator() *Validator {
	return &Validator{}
}

// Validate اعتبارسنجی کامل config
func (v *Validator) Validate(cfg *ServerConfig) error {
	// Version check
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version: %d (expected 1)", cfg.Version)
	}
	
	// Server info
	if err := v.validateServerInfo(&cfg.Server); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	
	// SNI
	if cfg.SNI.Enabled {
		if err := v.validateSNI(&cfg.SNI); err != nil {
			return fmt.Errorf("sni: %w", err)
		}
	}
	
	// Cover traffic
	if cfg.Cover.Enabled {
		if err := v.validateCover(&cfg.Cover); err != nil {
			return fmt.Errorf("cover: %w", err)
		}
	}
	
	// DNS fallback
	if cfg.DNS.Enabled {
		if err := v.validateDNS(&cfg.DNS); err != nil {
			return fmt.Errorf("dns: %w", err)
		}
	}
	
	// uTLS
	if cfg.UTLS.Enabled {
		if err := v.validateUTLS(&cfg.UTLS); err != nil {
			return fmt.Errorf("utls: %w", err)
		}
	}
	
	// Fragmentation
	if cfg.Fragment.Enabled {
		if err := v.validateFragment(&cfg.Fragment); err != nil {
			return fmt.Errorf("fragment: %w", err)
		}
	}
	
	return nil
}

// validateServerInfo اعتبارسنجی اطلاعات سرور
func (v *Validator) validateServerInfo(info *ServerInfo) error {
	if info.Name == "" {
		return fmt.Errorf("name is required")
	}
	
	if info.Address == "" {
		return fmt.Errorf("address is required")
	}
	
	// بررسی فرمت address
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
	
	// بررسی protocol
	validProtocols := map[string]bool{
		"guarch": true,
		"grouk":  true,
		"zhip":   true,
	}
	
	if !validProtocols[info.Protocol] {
		return fmt.Errorf("invalid protocol: %s (must be guarch/grouk/zhip)", info.Protocol)
	}
	
	// بررسی PSK
	if info.PSK == "" {
		return fmt.Errorf("PSK is required")
	}
	
	if len(info.PSK) < 16 {
		return fmt.Errorf("PSK too short (minimum 16 characters)")
	}
	
	return nil
}

// validateSNI اعتبارسنجی SNI config
func (v *Validator) validateSNI(sni *SNIConfig) error {
	if len(sni.Domains) == 0 {
		return fmt.Errorf("no domains specified")
	}
	
	// بررسی mode
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
	// ← حذف: hasFallback := false (استفاده نمیشد)
	
	for i, d := range sni.Domains {
		if d.Domain == "" {
			return fmt.Errorf("domain %d: empty domain", i)
		}
		
		// بررسی فرمت domain
		if !v.isValidDomain(d.Domain) {
			return fmt.Errorf("domain %d: invalid domain format: %s", i, d.Domain)
		}
		
		if d.Weight < 0 {
			return fmt.Errorf("domain %d: negative weight", i)
		}
		
		totalWeight += d.Weight
		
		// ← حذف: if d.Fallback { hasFallback = true }
	}
	
	// برای weighted mode، total weight باید > 0 باشه
	if sni.Mode == "weighted" && totalWeight == 0 {
		return fmt.Errorf("weighted mode requires at least one domain with weight > 0")
	}
	
	return nil
}

// validateCover اعتبارسنجی cover traffic config
func (v *Validator) validateCover(cover *CoverConfig) error {
	if len(cover.Domains) == 0 {
		return fmt.Errorf("no domains specified")
	}
	
	// بررسی mode
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
		
		// بررسی intervals
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

// validateDNS اعتبارسنجی DNS config
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

// validateUTLS اعتبارسنجی uTLS config
func (v *Validator) validateUTLS(utls *UTLSConfig) error {
	if utls.Fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}
	
	// می‌تونیم لیست fingerprint های معتبر رو چک کنیم
	// ولی الان فقط empty check می‌کنیم
	
	return nil
}

// validateFragment اعتبارسنجی fragmentation config
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

// isValidDomain بررسی معتبر بودن domain
func (v *Validator) isValidDomain(domain string) bool {
	// حذف www. اگه داره
	domain = strings.TrimPrefix(domain, "www.")
	
	// بررسی طول
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	
	// بررسی کاراکترهای مجاز
	for _, ch := range domain {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '.' || ch == '-') {
			return false
		}
	}
	
	// بررسی با url.Parse
	u, err := url.Parse("https://" + domain)
	if err != nil {
		return false
	}
	
	if u.Host != domain {
		return false
	}
	
	return true
}
