# https-dns-proxy
A Web interface to DNS.  


This is a quick implementation to be the opposite side of: https://github.com/wrouesnel/dns-over-https-proxy

You should be able to run this on a virtual instance where ever.  Or any CDN or Freedom advocate can run it.

This is a work in progress.  I used the basic TLS webserver from Go.  PRs and updates welcome.

/etc/dnsproxy.yaml

```
dnsserver: 8.8.8.8
dnsport: 53
listenport: 8415
sslkeypath:
sslcrtpath:
```

It currently has no logging.  Will probably add that shortly.  

All the heavy lifting is done with http://github.com/miekg/dns


Travis build status: [![Build Status](https://travis-ci.org/Harnish/https-dns-proxy.svg?branch=master)](https://travis-ci.org/Harnish/https-dns-proxy)
