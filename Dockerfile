FROM golang:1.27.1 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app .

FROM alpine:3.22
RUN adduser -D -H -u 10001 app
COPY --from=builder /app /usr/local/bin/https-dns-proxy
USER app
EXPOSE 8414
# plain-HTTP check; override with --health-cmd if running with TLS
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -q --spider http://127.0.0.1:8414/query || exit 1
ENTRYPOINT ["https-dns-proxy"]
