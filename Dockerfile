FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /guarch-server ./cmd/guarch-server/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /guarch-client ./cmd/guarch-client/

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

COPY --from=builder /guarch-server /usr/local/bin/guarch-server
COPY --from=builder /guarch-client /usr/local/bin/guarch-client
COPY configs/ /etc/guarch/

EXPOSE 8443 8080

ENTRYPOINT ["guarch-server"]
CMD ["-addr", ":8443", "-decoy", ":8080"]
