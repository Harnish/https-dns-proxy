# https-dns-proxy
A Web interface to DNS.  


This is a quick implementation to be the opposite side of: https://github.com/wrouesnel/dns-over-https-proxy

You should be able to run this on a virtual instance where ever.  Or any CDN or Freedom advocate can run it.

This is a work in progress.  I used the basic TLS webserver from Go.  PRs and updates welcome.

## Building
```
go build
```

## Running
```
./https-dns-proxy
```

## Usage 
```
Usage of ./https-dns-proxy:
  -conf string
    	Location of a config file.  Will override passed in parameters
  -dnsport string
    	Port on the DNS server to talk to (default "53")
  -dnsserver string
    	DNS server you want to use as your source (default "8.8.8.8")
  -log string
    	Directory for log file.  Will not log if param is missing
  -port string
    	Port you want to listen on (default "8414")
  -sslcrtpath string
    	Path to SSL CRT file
  -sslkeypath string
    	Path to SSL Key file
```


## Config file

/etc/dnsproxy.yaml

```
dnsserver: 8.8.8.8
dnsport: 53
listenport: 8415
sslkeypath:
sslcrtpath:
logpath:
```

If logpath is set it will create "dns-access.log" in that directory and log all requests there.

## API

`GET /resolve?name=<name>[&type=<n>][&dnssec=1][&cd=1]`

| Param | Meaning |
|---|---|
| `name` | Name to look up (required) |
| `type` | Numeric record type, e.g. `1` A, `28` AAAA, `15` MX. Default `255` (ANY) |
| `dnssec` | `1` sets the EDNS0 DO bit; RRSIGs are included in `Answer` |
| `cd` | `1` sets the Checking Disabled bit |

```
curl 'http://localhost:8414/resolve?name=example.com&type=1&dnssec=1'
```

The response is JSON: `Status` is the DNS rcode, `AD`/`CD`/`TC`/`RD`/`RA` come from the upstream response, and `Comment` explains failures (e.g. `NXDOMAIN`). `AD` is only meaningful if your upstream resolver validates DNSSEC. Truncated UDP answers are retried over TCP.

Errors are also JSON: `400` for a missing name or invalid type, `502` if the upstream DNS server can't be reached.

A human-friendly lookup page is at `/query`.

**Note:** there is no authentication or rate limiting, so anyone who can reach the server can use it as a resolver.

## Testing
```
go test ./...
```

All the heavy lifting is done with http://github.com/miekg/dns

GoDoc:  [![Godoc](https://godoc.org/github.com/Harnish/https-dns-proxy?status.png)](https://godoc.org/github.com/Harnish/https-dns-proxy)
