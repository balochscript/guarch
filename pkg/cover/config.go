package cover

import "time"

type DomainConfig struct {
	Domain      string
	Paths       []string
	Weight      int
	MinInterval time.Duration
	MaxInterval time.Duration
}

type Config struct {
	Enabled       bool
	Domains       []DomainConfig
	MaxConcurrent int
	IdleTraffic   bool
}

func DefaultConfig() *Config {
	return &Config{
		Enabled:       true,
		MaxConcurrent: 3,
		IdleTraffic:   true,
		Domains: []DomainConfig{
			{
				Domain:      "www.google.com",
				Paths:       []string{"/", "/search?q=weather", "/search?q=news", "/search?q=golang"},
				Weight:      30,
				MinInterval: 2 * time.Second,
				MaxInterval: 8 * time.Second,
			},
			{
				Domain:      "www.microsoft.com",
				Paths:       []string{"/", "/en-us", "/en-us/windows"},
				Weight:      20,
				MinInterval: 3 * time.Second,
				MaxInterval: 10 * time.Second,
			},
			{
				Domain:      "github.com",
				Paths:       []string{"/", "/explore", "/trending"},
				Weight:      15,
				MinInterval: 4 * time.Second,
				MaxInterval: 12 * time.Second,
			},
			{
				Domain:      "stackoverflow.com",
				Paths:       []string{"/", "/questions"},
				Weight:      15,
				MinInterval: 3 * time.Second,
				MaxInterval: 10 * time.Second,
			},
			{
				Domain:      "www.cloudflare.com",
				Paths:       []string{"/", "/learning"},
				Weight:      10,
				MinInterval: 5 * time.Second,
				MaxInterval: 15 * time.Second,
			},
			{
				Domain:      "learn.microsoft.com",
				Paths:       []string{"/", "/en-us/dotnet", "/en-us/azure"},
				Weight:      10,
				MinInterval: 4 * time.Second,
				MaxInterval: 12 * time.Second,
			},
		},
	}
}
