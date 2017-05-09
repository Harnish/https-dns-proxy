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
logpath
```

If logpath is set it will create "dns-access.log" in that directory and log all requests there.

All the heavy lifting is done with http://github.com/miekg/dns


Travis build status: [![Build Status](https://travis-ci.org/Harnish/https-dns-proxy.svg?branch=master)](https://travis-ci.org/Harnish/https-dns-proxy)
GoDoc:  [![Godoc](https://godoc.org/github.com/Harnish/https-dns-proxy?status.png)](https://godoc.org/github.com/Harnish/https-dns-proxy)
