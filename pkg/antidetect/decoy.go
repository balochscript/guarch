package antidetect

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

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

	time.Sleep(time.Duration(rand.Intn(100)+50) * time.Millisecond)

	w.Header().Set("Server", "nginx/1.24.0")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cache-Control", "public, max-age=3600")
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
<title>CloudFront CDN Services</title>
<style>
body{font-family:Arial,sans-serif;margin:40px;background:#f5f5f5}
.container{max-width:800px;margin:0 auto;background:#fff;padding:30px;border-radius:8px;box-shadow:0 2px 4px rgba(0,0,0,0.1)}
h1{color:#232f3e}
p{color:#666;line-height:1.6}
.footer{margin-top:40px;padding-top:20px;border-top:1px solid #eee;color:#999;font-size:14px}
</style>
</head>
<body>
<div class="container">
<h1>CloudFront CDN Services</h1>
<p>Welcome to our content delivery network. We provide fast and reliable content delivery for websites and applications worldwide.</p>
<p>Our network spans across %d+ locations globally, ensuring low latency and high availability for your content.</p>
<p>For more information about our services, please visit our <a href="/about">about page</a>.</p>
<div class="footer">
<p>&copy; %d CloudFront CDN Services. All rights reserved.</p>
</div>
</div>
</body>
</html>`, rand.Intn(50)+200, time.Now().Year())
}

func (ds *DecoyServer) generateAboutPage() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>About - CloudFront CDN</title>
<style>
body{font-family:Arial,sans-serif;margin:40px;background:#f5f5f5}
.container{max-width:800px;margin:0 auto;background:#fff;padding:30px;border-radius:8px;box-shadow:0 2px 4px rgba(0,0,0,0.1)}
h1{color:#232f3e}
p{color:#666;line-height:1.6}
</style>
</head>
<body>
<div class="container">
<h1>About Us</h1>
<p>CloudFront CDN has been providing content delivery services since 2015.</p>
<p>We currently serve over %d million requests per day.</p>
<p><a href="/">Back to Home</a></p>
</div>
</body>
</html>`, rand.Intn(500)+100)
}

func (ds *DecoyServer) generateContactPage() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Contact - CloudFront CDN</title>
<style>
body{font-family:Arial,sans-serif;margin:40px;background:#f5f5f5}
.container{max-width:800px;margin:0 auto;background:#fff;padding:30px;border-radius:8px;box-shadow:0 2px 4px rgba(0,0,0,0.1)}
h1{color:#232f3e}
p{color:#666;line-height:1.6}
</style>
</head>
<body>
<div class="container">
<h1>Contact Us</h1>
<p>Email: support@cloudfront-cdn.example.com</p>
<p><a href="/">Back to Home</a></p>
</div>
</body>
</html>`
}

func (ds *DecoyServer) generateBlogPage() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Blog - CloudFront CDN</title>
<style>
body{font-family:Arial,sans-serif;margin:40px;background:#f5f5f5}
.container{max-width:800px;margin:0 auto;background:#fff;padding:30px;border-radius:8px;box-shadow:0 2px 4px rgba(0,0,0,0.1)}
h1{color:#232f3e}
p{color:#666;line-height:1.6}
.post{margin-bottom:30px;padding-bottom:20px;border-bottom:1px solid #eee}
</style>
</head>
<body>
<div class="container">
<h1>Blog</h1>
<div class="post">
<h2>Improving Cache Hit Ratios</h2>
<p>Published: %s</p>
<p>Learn how to optimize your CDN configuration.</p>
</div>
<p><a href="/">Back to Home</a></p>
</div>
</body>
</html>`,
		time.Now().AddDate(0, 0, -rand.Intn(30)).Format("January 2, 2006"),
	)
}

func (ds *DecoyServer) generate404() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>404 Not Found</title>
<style>
body{font-family:Arial,sans-serif;margin:40px;text-align:center;background:#f5f5f5}
h1{color:#232f3e;font-size:72px;margin-bottom:0}
p{color:#666}
</style>
</head>
<body>
<h1>404</h1>
<p>The requested resource was not found.</p>
<p><a href="/">Return to homepage</a></p>
</body>
</html>`
}
