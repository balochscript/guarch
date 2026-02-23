.PHONY: build test clean client server zhip-client zhip-server

# ═══ Guarch (TCP — Stealth) ═══
build: client server

client:
	go build -o bin/guarch-client ./cmd/guarch-client/

server:
	go build -o bin/guarch-server ./cmd/guarch-server/

# ═══ Zhip (QUIC — Fast) ═══
zhip-client:
	go build -o bin/zhip-client ./cmd/zhip-client/

zhip-server:
	go build -o bin/zhip-server ./cmd/zhip-server/

zhip: zhip-client zhip-server

# ═══ All Protocols ═══
all: build zhip

# ═══ Test ═══
test:
	go test ./pkg/... -v -count=1

test-race:
	go test ./pkg/... -race -count=1

# ═══ Tools ═══
clean:
	rm -rf bin/

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

# ═══ Cross Compile — Guarch ═══
linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/guarch-client-linux-amd64 ./cmd/guarch-client/
	GOOS=linux GOARCH=amd64 go build -o bin/guarch-server-linux-amd64 ./cmd/guarch-server/

linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o bin/guarch-client-linux-arm64 ./cmd/guarch-client/
	GOOS=linux GOARCH=arm64 go build -o bin/guarch-server-linux-arm64 ./cmd/guarch-server/

windows:
	GOOS=windows GOARCH=amd64 go build -o bin/guarch-client.exe ./cmd/guarch-client/

darwin:
	GOOS=darwin GOARCH=arm64 go build -o bin/guarch-client-darwin ./cmd/guarch-client/

# ═══ Cross Compile — Zhip ═══
zhip-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/zhip-client-linux-amd64 ./cmd/zhip-client/
	GOOS=linux GOARCH=amd64 go build -o bin/zhip-server-linux-amd64 ./cmd/zhip-server/

zhip-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o bin/zhip-client-linux-arm64 ./cmd/zhip-client/
	GOOS=linux GOARCH=arm64 go build -o bin/zhip-server-linux-arm64 ./cmd/zhip-server/

zhip-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/zhip-client.exe ./cmd/zhip-client/

zhip-darwin:
	GOOS=darwin GOARCH=arm64 go build -o bin/zhip-client-darwin ./cmd/zhip-client/

# ═══ All Platforms — All Protocols ═══
all-platforms: linux-amd64 linux-arm64 windows darwin zhip-linux-amd64 zhip-linux-arm64 zhip-windows zhip-darwin
