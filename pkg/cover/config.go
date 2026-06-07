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
	if c.MaxConcurrent == 0 {
		c.MaxConcurrent = 3
	}
	
	for i := range c.Domains {
		if c.Domains[i].Weight == 0 {
			c.Domains[i].Weight = 10
		}
		if c.Domains[i].MinInterval == 0 {
			c.Domains[i].MinInterval = 2 * time.Second
		}
		if c.Domains[i].MaxInterval == 0 {
			c.Domains[i].MaxInterval = 10 * time.Second
		}
	}
}
