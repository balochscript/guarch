package cover

import (
	"fmt"
	"time"
)

type DomainConfig struct {
	Domain      string        `json:"domain"`
	Paths       []string      `json:"paths"`
	Weight      int           `json:"weight"`
	MinInterval time.Duration `json:"min_interval"`
	MaxInterval time.Duration `json:"max_interval"`
}

type Config struct {
	Enabled       bool           `json:"enabled"`
	Domains       []DomainConfig `json:"domains"`
	MaxConcurrent int            `json:"max_concurrent"`
	IdleTraffic   bool           `json:"idle_traffic"`
	TransportHost string         `json:"-"`
}

type ModeConfig struct {
	MaxPadding       int
	BatteryThreshold int
	HysteresisDelay  time.Duration
	IdleThreshold    int64
	LightThreshold   int64
	MediumThreshold  int64
	HeavyThreshold   int64
}

func NewConfig() *Config {
	return &Config{
		Enabled:       false,
		Domains:       []DomainConfig{},
		MaxConcurrent: 3,
		IdleTraffic:   true,
	}
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if len(c.Domains) == 0 {
		return fmt.Errorf("cover: no domains configured")
	}
	for i, d := range c.Domains {
		if d.Domain == "" {
			return fmt.Errorf("cover: domain %d is empty", i)
		}
		if len(d.Paths) == 0 {
			return fmt.Errorf("cover: domain %d has no paths", i)
		}
		if d.Weight < 0 {
			return fmt.Errorf("cover: domain %d has negative weight", i)
		}
	}
	return nil
}

func (c *Config) ApplyDefaults() {
	if c.MaxConcurrent <= 0 {
		c.MaxConcurrent = 3
	}
	if c.MaxConcurrent > 10 {
		c.MaxConcurrent = 10
	}
	for i := range c.Domains {
		if c.Domains[i].Weight <= 0 {
			c.Domains[i].Weight = 10
		}
		if c.Domains[i].MinInterval <= 0 {
			c.Domains[i].MinInterval = 2 * time.Second
		}
		if c.Domains[i].MaxInterval <= 0 {
			c.Domains[i].MaxInterval = 10 * time.Second
		}
		if c.Domains[i].MaxInterval < c.Domains[i].MinInterval {
			c.Domains[i].MaxInterval = c.Domains[i].MinInterval + 5*time.Second
		}
	}
}

func ParseBytesPerMin(s string) int64 {
	if s == "" {
		return 0
	}
	var val int64
	var unit string
	_, err := fmt.Sscanf(s, "%d%s", &val, &unit)
	if err != nil {
		return 0
	}
	switch unit {
	case "B/min", "B", "b/min", "":
		return val
	case "KB/min", "KB", "K", "kB/min":
		return val * 1000
	case "MB/min", "MB", "M", "mB/min":
		return val * 1000 * 1000
	case "GB/min", "GB", "G":
		return val * 1000 * 1000 * 1000
	default:
		return val * 1000
	}
}
