.PHONY: build test clean client server

build: client server

client:
	go build -o bin/guarch-client ./cmd/guarch-client/

server:
	go build -o bin/guarch-server ./cmd/guarch-server/

test:
	go test ./pkg/... -v -count=1

test-race:
	go test ./pkg/... -race -count=1

clean:
	rm -rf bin/

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

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

all-platforms: linux-amd64 linux-arm64 windows darwin
