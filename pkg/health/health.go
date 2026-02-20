package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

type Checker struct {
	startTime     time.Time
	activeConns   atomic.Int64
	totalConns    atomic.Int64
	totalBytes    atomic.Int64
	coverRequests atomic.Int64
	errors        atomic.Int64
}

func New() *Checker {
	return &Checker{
		startTime: time.Now(),
	}
}

func (c *Checker) AddConn() {
	c.activeConns.Add(1)
	c.totalConns.Add(1)
}

func (c *Checker) RemoveConn() {
	c.activeConns.Add(-1)
}

func (c *Checker) AddBytes(n int64) {
	c.totalBytes.Add(n)
}

func (c *Checker) AddCoverRequest() {
	c.coverRequests.Add(1)
}

func (c *Checker) AddError() {
	c.errors.Add(1)
}

type Status struct {
	Status        string `json:"status"`
	Uptime        string `json:"uptime"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	ActiveConns   int64  `json:"active_connections"`
	TotalConns    int64  `json:"total_connections"`
	TotalBytes    int64  `json:"total_bytes"`
	CoverRequests int64  `json:"cover_requests"`
	Errors        int64  `json:"errors"`
	GoRoutines    int    `json:"goroutines"`
	MemoryMB      uint64 `json:"memory_mb"`
}

func (c *Checker) GetStatus() Status {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	uptime := time.Since(c.startTime)

	return Status{
		Status:        "running",
		Uptime:        formatDuration(uptime),
		UptimeSeconds: int64(uptime.Seconds()),
		ActiveConns:   c.activeConns.Load(),
		TotalConns:    c.totalConns.Load(),
		TotalBytes:    c.totalBytes.Load(),
		CoverRequests: c.coverRequests.Load(),
		Errors:        c.errors.Load(),
		GoRoutines:    runtime.NumGoroutine(),
		MemoryMB:      mem.Alloc / 1024 / 1024,
	}
}

func (c *Checker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status := c.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(status)
}

func (c *Checker) StartServer(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		c.ServeHTTP(w, r)
	})
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	go http.ListenAndServe(addr, mux)
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
