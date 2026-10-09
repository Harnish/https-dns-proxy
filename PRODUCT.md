# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users
Self-hosters and operators who run the service (on a VM or CDN edge), plus the people and clients who consume its `/resolve` API where plain UDP/53 DNS is unavailable or blocked. Operators also use the `/query` page to check records by hand.

## Product Purpose
A web interface to DNS: accepts a name and record type over HTTP(S), queries an upstream DNS server, and returns a JSON answer. Success is a tiny, reliable service that stays up under bad input and is trivial to deploy.

## Positioning
Tiny, drop-in server side: a single Go binary that is the server counterpart to https://github.com/wrouesnel/dns-over-https-proxy, exposing a Google-style JSON resolve API.

## Operating Context
Runs as a single binary, systemd service (deb/rpm via nfpm), or container. Configured by flags or a YAML file (`dnsserver`, `dnsport`, `listenport`, `sslkeypath`, `sslcrtpath`, `logpath`). Optional TLS; plain HTTP by default.

## Capabilities and Constraints
- Endpoints: `/` (redirect), `/query` (HTML lookup page), `/resolve?name=&type=` (JSON).
- Lookup page is a single HTML string embedded in `webpage.go`; no framework, no build step.
- UDP-only upstream queries; the response format is not RFC 8484 wire format.
- Open resolver with no auth or rate limiting. Undecided.

## Brand Commitments
Name: https-dns-proxy. No other brand assets.

## Evidence on Hand
README and source only. No testimonials, benchmarks, or customers exist; do not invent any.

## Product Principles
- Stay small: one binary, minimal dependencies.
- Never crash on user input.
- Treat every query parameter as hostile.
- Plain HTTP works out of the box; TLS is opt-in.
