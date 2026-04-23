# Makefile - Guarch Protocol Suite
# Version: 1.0.1

.PHONY: build test clean client server zhip-client zhip-server grouk-client grouk-server
.PHONY: all run-client run-server install fmt vet tidy
.PHONY: linux-amd64 linux-arm64 windows darwin all-platforms

# ═══════════════════════════════════════════════════════════
# Build Info
# ═══════════════════════════════════════════════════════════

VERSION := 1.0.1
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -s -w \
	-X 'main.version=$(VERSION)' \
	-X 'main.buildTime=$(BUILD_TIME)' \
	-X 'main.gitCommit=$(GIT_COMMIT)' \
	-X 'main.gitBranch=$(GIT_BRANCH)'

# ═══════════════════════════════════════════════════════════
# Guarch 🏹 (TCP — Stealth)
# ═══════════════════════════════════════════════════════════

build: client server

client:
	@echo "🔨 Building Guarch Client v$(VERSION)..."
	@go build -ldflags "$(LDFLAGS)" -o bin/guarch-client ./cmd/guarch-client/
	@echo "✅ bin/guarch-client"

server:
	@echo "🔨 Building Guarch Server v$(VERSION)..."
	@go build -ldflags "$(LDFLAGS)" -o bin/guarch-server ./cmd/guarch-server/
	@echo "✅ bin/guarch-server"

# ═══════════════════════════════════════════════════════════
# Zhip ⚡ (QUIC — Fast)
# ═══════════════════════════════════════════════════════════

zhip-client:
	@echo "🔨 Building Zhip Client..."
	@go build -ldflags "$(LDFLAGS)" -o bin/zhip-client ./cmd/zhip-client/
	@echo "✅ bin/zhip-client"

zhip-server:
	@echo "🔨 Building Zhip Server..."
	@go build -ldflags "$(LDFLAGS)" -o bin/zhip-server ./cmd/zhip-server/
	@echo "✅ bin/zhip-server"

zhip: zhip-client zhip-server

# ═══════════════════════════════════════════════════════════
# Grouk 🌩️ (Raw UDP — Ultra Fast)
# ═══════════════════════════════════════════════════════════

grouk-client:
	@echo "🔨 Building Grouk Client..."
	@go build -ldflags "$(LDFLAGS)" -o bin/grouk-client ./cmd/grouk-client/
	@echo "✅ bin/grouk-client"

grouk-server:
	@echo "🔨 Building Grouk Server..."
	@go build -ldflags "$(LDFLAGS)" -o bin/grouk-server ./cmd/grouk-server/
	@echo "✅ bin/grouk-server"

grouk: grouk-client grouk-server

# ═══════════════════════════════════════════════════════════
# All Protocols
# ═══════════════════════════════════════════════════════════

all: build zhip grouk
	@echo ""
	@echo "✅ All protocols built successfully!"
	@echo "   Guarch: bin/guarch-{client,server}"
	@echo "   Zhip:   bin/zhip-{client,server}"
	@echo "   Grouk:  bin/grouk-{client,server}"

# ═══════════════════════════════════════════════════════════
# Run & Test
# ═══════════════════════════════════════════════════════════

run-client:
	@echo "🚀 Running Guarch Client..."
	./bin/guarch-client -config configs/example_client.json

run-server:
	@echo "🚀 Running Guarch Server..."
	./bin/guarch-server -config configs/example_server.json

test:
	@echo "🧪 Running tests..."
	@go test ./pkg/... -v -count=1

test-race:
	@echo "🧪 Running tests with race detector..."
	@go test ./pkg/... -race -count=1

test-coverage:
	@echo "🧪 Running tests with coverage..."
	@go test ./pkg/... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

bench:
	@echo "⚡ Running benchmarks..."
	@go test ./pkg/... -bench=. -benchmem

# ═══════════════════════════════════════════════════════════
# Development Tools
# ═══════════════════════════════════════════════════════════

fmt:
	@echo "🎨 Formatting code..."
	@go fmt ./...

vet:
	@echo "🔍 Running go vet..."
	@go vet ./...

tidy:
	@echo "📦 Tidying dependencies..."
	@go mod tidy

lint: fmt vet
	@echo "✅ Code linting complete"

install:
	@echo "📥 Installing binaries..."
	@go install -ldflags "$(LDFLAGS)" ./cmd/guarch-client/
	@go install -ldflags "$(LDFLAGS)" ./cmd/guarch-server/
	@echo "✅ Installed to $(shell go env GOPATH)/bin/"

clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf bin/ coverage.out coverage.html
	@echo "✅ Clean complete"

# ═══════════════════════════════════════════════════════════
# Cross Compile — Guarch
# ═══════════════════════════════════════════════════════════

linux-amd64:
	@echo "🔨 Building for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-client-linux-amd64 ./cmd/guarch-client/
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-server-linux-amd64 ./cmd/guarch-server/
	@echo "✅ bin/guarch-{client,server}-linux-amd64"

linux-arm64:
	@echo "🔨 Building for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-client-linux-arm64 ./cmd/guarch-client/
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-server-linux-arm64 ./cmd/guarch-server/
	@echo "✅ bin/guarch-{client,server}-linux-arm64"

windows:
	@echo "🔨 Building for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-client.exe ./cmd/guarch-client/
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-server.exe ./cmd/guarch-server/
	@echo "✅ bin/guarch-{client,server}.exe"

darwin:
	@echo "🔨 Building for macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-client-darwin-arm64 ./cmd/guarch-client/
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-client-darwin-amd64 ./cmd/guarch-client/
	@echo "✅ bin/guarch-client-darwin-{arm64,amd64}"

# ═══════════════════════════════════════════════════════════
# Cross Compile — Zhip
# ═══════════════════════════════════════════════════════════

zhip-linux-amd64:
	@echo "🔨 Building Zhip for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/zhip-client-linux-amd64 ./cmd/zhip-client/
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/zhip-server-linux-amd64 ./cmd/zhip-server/
	@echo "✅ bin/zhip-{client,server}-linux-amd64"

zhip-linux-arm64:
	@echo "🔨 Building Zhip for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/zhip-client-linux-arm64 ./cmd/zhip-client/
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/zhip-server-linux-arm64 ./cmd/zhip-server/
	@echo "✅ bin/zhip-{client,server}-linux-arm64"

zhip-windows:
	@echo "🔨 Building Zhip for Windows..."
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/zhip-client.exe ./cmd/zhip-client/
	@echo "✅ bin/zhip-client.exe"

zhip-darwin:
	@echo "🔨 Building Zhip for macOS..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/zhip-client-darwin-arm64 ./cmd/zhip-client/
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/zhip-client-darwin-amd64 ./cmd/zhip-client/
	@echo "✅ bin/zhip-client-darwin-{arm64,amd64}"

# ═══════════════════════════════════════════════════════════
# Cross Compile — Grouk
# ═══════════════════════════════════════════════════════════

grouk-linux-amd64:
	@echo "🔨 Building Grouk for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/grouk-client-linux-amd64 ./cmd/grouk-client/
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/grouk-server-linux-amd64 ./cmd/grouk-server/
	@echo "✅ bin/grouk-{client,server}-linux-amd64"

grouk-linux-arm64:
	@echo "🔨 Building Grouk for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/grouk-client-linux-arm64 ./cmd/grouk-client/
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/grouk-server-linux-arm64 ./cmd/grouk-server/
	@echo "✅ bin/grouk-{client,server}-linux-arm64"

grouk-windows:
	@echo "🔨 Building Grouk for Windows..."
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/grouk-client.exe ./cmd/grouk-client/
	@echo "✅ bin/grouk-client.exe"

grouk-darwin:
	@echo "🔨 Building Grouk for macOS..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/grouk-client-darwin-arm64 ./cmd/grouk-client/
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/grouk-client-darwin-amd64 ./cmd/grouk-client/
	@echo "✅ bin/grouk-client-darwin-{arm64,amd64}"

# ═══════════════════════════════════════════════════════════
# All Platforms — All Protocols
# ═══════════════════════════════════════════════════════════

all-platforms: linux-amd64 linux-arm64 windows darwin \
               zhip-linux-amd64 zhip-linux-arm64 zhip-windows zhip-darwin \
               grouk-linux-amd64 grouk-linux-arm64 grouk-windows grouk-darwin
	@echo ""
	@echo "🎉 All platforms built successfully!"
	@echo ""
	@ls -lh bin/

# ═══════════════════════════════════════════════════════════
# Release Build
# ═══════════════════════════════════════════════════════════

release: clean all-platforms
	@echo ""
	@echo "📦 Creating release archives..."
	@mkdir -p release
	@cd bin && tar -czf ../release/guarch-$(VERSION)-linux-amd64.tar.gz guarch-*-linux-amd64
	@cd bin && tar -czf ../release/guarch-$(VERSION)-linux-arm64.tar.gz guarch-*-linux-arm64
	@cd bin && zip -q ../release/guarch-$(VERSION)-windows-amd64.zip guarch-*.exe
	@cd bin && tar -czf ../release/guarch-$(VERSION)-darwin.tar.gz guarch-*-darwin-*
	@echo "✅ Release archives in release/"
	@ls -lh release/

# ═══════════════════════════════════════════════════════════
# Docker
# ═══════════════════════════════════════════════════════════

docker-build:
	@echo "🐳 Building Docker image..."
	@docker build -t guarch:$(VERSION) -t guarch:latest .
	@echo "✅ Docker image: guarch:$(VERSION)"

docker-run-server:
	@echo "🐳 Running Guarch Server in Docker..."
	@docker run -d -p 8443:8443 -p 8080:8080 \
		--name guarch-server \
		guarch:latest \
		-psk "$(PSK)" -mode stealth

docker-stop:
	@docker stop guarch-server
	@docker rm guarch-server

# ═══════════════════════════════════════════════════════════
# Help
# ═══════════════════════════════════════════════════════════

help:
	@echo "Guarch Protocol Suite - Makefile v$(VERSION)"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build           Build Guarch client + server"
	@echo "  make zhip            Build Zhip client + server"
	@echo "  make grouk           Build Grouk client + server"
	@echo "  make all             Build all protocols"
	@echo ""
	@echo "Run Commands:"
	@echo "  make run-client      Run Guarch client"
	@echo "  make run-server      Run Guarch server"
	@echo ""
	@echo "Test Commands:"
	@echo "  make test            Run all tests"
	@echo "  make test-race       Run tests with race detector"
	@echo "  make test-coverage   Generate coverage report"
	@echo "  make bench           Run benchmarks"
	@echo ""
	@echo "Development:"
	@echo "  make fmt             Format code"
	@echo "  make vet             Run go vet"
	@echo "  make lint            Run fmt + vet"
	@echo "  make tidy            Tidy dependencies"
	@echo "  make clean           Remove build artifacts"
	@echo ""
	@echo "Cross-Compile:"
	@echo "  make linux-amd64     Build for Linux x64"
	@echo "  make linux-arm64     Build for Linux ARM64"
	@echo "  make windows         Build for Windows"
	@echo "  make darwin          Build for macOS"
	@echo "  make all-platforms   Build for all platforms"
	@echo ""
	@echo "Release:"
	@echo "  make release         Create release archives"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build    Build Docker image"
	@echo "  make docker-run-server  Run server in Docker"
	@echo ""
	@echo "Version: $(VERSION) ($(GIT_BRANCH)@$(GIT_COMMIT))"
