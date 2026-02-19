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
				Paths:       []string{"/", "/search?q=weather", "/search?q=news"},
				Weight:      30,
				MinInterval: 2 * time.Second,
				MaxInterval: 8 * time.Second,
			},
			{
				Domain:      "www.microsoft.com",
				Paths:       []string{"/", "/en-us"},
				Weight:      20,
				MinInterval: 3 * time.Second,
				MaxInterval: 10 * time.Second,
			},
			{
				Domain:      "www.github.com",
				Paths:       []string{"/", "/explore"},
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
				Domain:      "www.wikipedia.org",
				Paths:       []string{"/"},
				Weight:      10,
				MinInterval: 5 * time.Second,
				MaxInterval: 15 * time.Second,
			},
			{
				Domain:      "www.amazon.com",
				Paths:       []string{"/", "/s?k=books"},
				Weight:      10,
				MinInterval: 4 * time.Second,
				MaxInterval: 12 * time.Second,
			},
		},
	}
}
