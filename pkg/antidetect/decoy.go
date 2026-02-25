package antidetect

import (
	crand "crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"
)

func cryptoRandIntn(n int) int {
	if n <= 0 {
		return 0
	}
	val, err := crand.Int(crand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return n / 2
	}
	return int(val.Int64())
}

type DecoyServer struct {
	pages map[string]string
}

func NewDecoyServer() *DecoyServer {
	ds := &DecoyServer{
		pages: make(map[string]string),
	}
	ds.pages["/"] = ds.GenerateHomePage()
	ds.pages["/about"] = ds.generateAboutPage()
	ds.pages["/contact"] = ds.generateContactPage()
	ds.pages["/blog"] = ds.generateBlogPage()
	return ds
}

func (ds *DecoyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[decoy] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

	delay := time.Duration(cryptoRandIntn(100)+50) * time.Millisecond
	time.Sleep(delay)

	w.Header().Set("Server", "nginx/1.24.0")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	// ✅ فیکس: Date رو دستی ست کن چون httptest خودش اضافه نمیکنه
	w.Header().Set("Date", time.Now().UTC().Format(http.TimeFormat))

	page, ok := ds.pages[r.URL.Path]
	if !ok {
		w.WriteHeader(404)
		w.Write([]byte(ds.generate404()))
		return
	}

	w.WriteHeader(200)
	w.Write([]byte(page))
}

func (ds *DecoyServer) GenerateHomePage() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>FastEdge CDN - Content Delivery Network</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;margin:0;background:#f8f9fa}
.nav{background:#1a1a2e;padding:15px 40px;color:#fff;display:flex;justify-content:space-between;align-items:center}
.nav a{color:#e0e0e0;text-decoration:none;margin-left:20px}
.container{max-width:900px;margin:40px auto;padding:30px}
.hero{background:#fff;padding:50px;border-radius:12px;box-shadow:0 2px 8px rgba(0,0,0,0.08);text-align:center;margin-bottom:30px}
h1{color:#1a1a2e;font-size:2.2em;margin-bottom:10px}
.subtitle{color:#666;font-size:1.1em;margin-bottom:30px}
.stats{display:flex;justify-content:space-around;margin:40px 0}
.stat{text-align:center}
.stat-num{font-size:2em;color:#4361ee;font-weight:bold}
.stat-label{color:#888;margin-top:5px}
.footer{text-align:center;padding:30px;color:#999;font-size:14px}
</style>
</head>
<body>
<div class="nav">
<strong>FastEdge CDN</strong>
<div><a href="/">Home</a><a href="/about">About</a><a href="/blog">Blog</a><a href="/contact">Contact</a></div>
</div>
<div class="container">
<div class="hero">
<h1>Global Content Delivery</h1>
<p class="subtitle">Fast, reliable, and secure content delivery for modern applications.</p>
<div class="stats">
<div class="stat"><div class="stat-num">%d+</div><div class="stat-label">Edge Locations</div></div>
<div class="stat"><div class="stat-num">%d+</div><div class="stat-label">Countries</div></div>
<div class="stat"><div class="stat-num">99.9%%%%</div><div class="stat-label">Uptime SLA</div></div>
</div>
</div>
</div>
<div class="footer">&copy; %d FastEdge CDN. All rights reserved.</div>
</body>
</html>`, cryptoRandIntn(100)+150, cryptoRandIntn(30)+40, time.Now().Year())
}

func (ds *DecoyServer) generateAboutPage() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><title>About - FastEdge CDN</title>
<style>body{font-family:-apple-system,sans-serif;margin:40px;background:#f8f9fa}.c{max-width:800px;margin:0 auto;background:#fff;padding:40px;border-radius:12px;box-shadow:0 2px 8px rgba(0,0,0,0.08)}h1{color:#1a1a2e}p{color:#555;line-height:1.8}a{color:#4361ee}</style>
</head><body><div class="c">
<h1>About FastEdge</h1>
<p>Founded in 2018, FastEdge CDN delivers content to over %d million users daily across our global network.</p>
<p>Our infrastructure is built on modern edge computing principles, ensuring your content reaches users with minimal latency.</p>
<p><a href="/">← Back to Home</a></p>
</div></body></html>`, cryptoRandIntn(500)+100)
}

func (ds *DecoyServer) generateContactPage() string {
	return `<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><title>Contact - FastEdge CDN</title>
<style>body{font-family:-apple-system,sans-serif;margin:40px;background:#f8f9fa}.c{max-width:800px;margin:0 auto;background:#fff;padding:40px;border-radius:12px;box-shadow:0 2px 8px rgba(0,0,0,0.08)}h1{color:#1a1a2e}p{color:#555;line-height:1.8}a{color:#4361ee}</style>
</head><body><div class="c">
<h1>Contact Us</h1>
<p>Enterprise Sales: sales@fastedge-cdn.example.com</p>
<p>Technical Support: support@fastedge-cdn.example.com</p>
<p>Status Page: status.fastedge-cdn.example.com</p>
<p><a href="/">← Back to Home</a></p>
</div></body></html>`
}

func (ds *DecoyServer) generateBlogPage() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><title>Blog - FastEdge CDN</title>
<style>body{font-family:-apple-system,sans-serif;margin:40px;background:#f8f9fa}.c{max-width:800px;margin:0 auto;background:#fff;padding:40px;border-radius:12px;box-shadow:0 2px 8px rgba(0,0,0,0.08)}h1{color:#1a1a2e}h2{color:#333}p{color:#555;line-height:1.8}.post{border-bottom:1px solid #eee;padding:20px 0}.date{color:#999;font-size:14px}a{color:#4361ee}</style>
</head><body><div class="c">
<h1>Engineering Blog</h1>
<div class="post"><h2>Optimizing TLS 1.3 Handshakes at Scale</h2><p class="date">%s</p><p>How we reduced connection setup time by 40%%%% across our edge network.</p></div>
<div class="post"><h2>Building a Global Anycast Network</h2><p class="date">%s</p><p>Lessons learned from deploying across 200+ locations worldwide.</p></div>
<p><a href="/">← Back to Home</a></p>
</div></body></html>`,
		time.Now().AddDate(0, 0, -cryptoRandIntn(14)).Format("January 2, 2006"),
		time.Now().AddDate(0, 0, -cryptoRandIntn(30)-14).Format("January 2, 2006"),
	)
}

func (ds *DecoyServer) generate404() string {
	return `<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><title>404 - FastEdge CDN</title>
<style>body{font-family:-apple-system,sans-serif;margin:0;display:flex;align-items:center;justify-content:center;min-height:100vh;background:#f8f9fa}
.c{text-align:center}h1{font-size:80px;color:#ddd;margin:0}p{color:#888}a{color:#4361ee}</style>
</head><body><div class="c"><h1>404</h1><p>The requested resource was not found on this server.</p><p><a href="/">Return to homepage</a></p></div></body></html>`
}
