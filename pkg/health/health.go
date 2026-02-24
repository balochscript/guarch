package health

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
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

func (c *Checker) AddConn()         { c.activeConns.Add(1); c.totalConns.Add(1) }
func (c *Checker) RemoveConn()      { c.activeConns.Add(-1) }
func (c *Checker) AddBytes(n int64) { c.totalBytes.Add(n) }
func (c *Checker) AddCoverRequest() { c.coverRequests.Add(1) }
func (c *Checker) AddError()        { c.errors.Add(1) }

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

// ✅ M21: StartServer حالا server برمیگردونه + خطای startup رو detect میکنه
// قبلاً: fire-and-forget → خطای port conflict نادیده گرفته میشد
// الان: listener اول ساخته میشه → خطا فوری برمیگرده + server برای shutdown
func (c *Checker) StartServer(addr string, authToken ...string) (*http.Server, error) {
	mux := http.NewServeMux()

	var token string
	if len(authToken) > 0 {
		token = authToken[0]
	}

	authMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if token != "" {
				got := r.Header.Get("Authorization")
				if subtle.ConstantTimeCompare([]byte(got), []byte("Bearer "+token)) != 1 {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			}
			next(w, r)
		}
	}

	mux.HandleFunc("/health", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		c.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ✅ M21: listener اول بساز تا port conflict فوری تشخیص داده بشه
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("health: listen %s: %w", addr, err)
	}

	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("[health] server error: %v", err)
		}
	}()

	log.Printf("[health] server started on %s", addr)
	return server, nil
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
