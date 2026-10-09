package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/miekg/dns"
)

// startUpstream runs a fake DNS server on UDP+TCP and points config at it.
func startUpstream(t *testing.T) {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := pc.LocalAddr().String()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	h := dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		resp := new(dns.Msg)
		resp.SetReply(req)
		name := req.Question[0].Name
		switch name {
		case "nx.test.":
			resp.Rcode = dns.RcodeNameError
		case "big.test.":
			if _, udp := w.RemoteAddr().(*net.UDPAddr); udp {
				resp.Truncated = true
				break
			}
			fallthrough
		default:
			rr, _ := dns.NewRR(name + " 60 IN A 1.2.3.4")
			resp.Answer = append(resp.Answer, rr)
			if o := req.IsEdns0(); o != nil && o.Do() {
				resp.AuthenticatedData = true
			}
		}
		w.WriteMsg(resp)
	})
	udp := &dns.Server{PacketConn: pc, Handler: h}
	tcp := &dns.Server{Listener: ln, Handler: h}
	go udp.ActivateAndServe()
	go tcp.ActivateAndServe()
	t.Cleanup(func() { udp.Shutdown(); tcp.Shutdown() })
	host, port, _ := net.SplitHostPort(addr)
	config = &Config{DNSServer: host, DNSPort: port}
}

type result struct {
	AD, CD, TC bool
	Status     int
	Comment    string
	Answer     []json.RawMessage
}

func query(t *testing.T, qs string) (int, result) {
	t.Helper()
	rec := httptest.NewRecorder()
	ResolveDNS(rec, httptest.NewRequest("GET", "/resolve?"+qs, nil))
	var out result
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body not JSON: %q: %v", rec.Body.String(), err)
	}
	return rec.Code, out
}

func TestResolve(t *testing.T) {
	startUpstream(t)
	tests := []struct {
		name, qs string
		code     int
		status   int
		comment  string
		ad       bool
		answers  int
	}{
		{"ok", "name=ok.test&type=1", 200, 0, "", false, 1},
		{"nxdomain does not crash", "name=nx.test&type=1", 200, dns.RcodeNameError, "NXDOMAIN", false, 0},
		{"bad type", "name=ok.test&type=zz", 400, dns.RcodeFormatError, "invalid type", false, 0},
		{"type out of range", "name=ok.test&type=70000", 400, dns.RcodeFormatError, "invalid type", false, 0},
		{"missing name", "type=1", 400, dns.RcodeFormatError, "missing name", false, 0},
		{"dnssec sets DO", "name=ok.test&type=1&dnssec=1", 200, 0, "", true, 1},
		{"truncated retries over tcp", "name=big.test&type=1", 200, 0, "", false, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, got := query(t, tc.qs)
			if code != tc.code || got.Status != tc.status || got.Comment != tc.comment || got.AD != tc.ad || len(got.Answer) != tc.answers {
				t.Errorf("got code=%d status=%d comment=%q AD=%v answers=%d", code, got.Status, got.Comment, got.AD, len(got.Answer))
			}
			if got.TC {
				t.Error("TC should be false after tcp retry")
			}
		})
	}
}

func TestUpstreamDown(t *testing.T) {
	l, _ := net.ListenPacket("udp", "127.0.0.1:0")
	host, port, _ := net.SplitHostPort(l.LocalAddr().String())
	l.Close()
	config = &Config{DNSServer: host, DNSPort: port}
	code, got := query(t, "name=ok.test&type=1")
	if code != http.StatusBadGateway || got.Status != dns.RcodeServerFailure || got.Comment == "" {
		t.Errorf("got code=%d status=%d comment=%q", code, got.Status, got.Comment)
	}
}

func TestPickUpstream(t *testing.T) {
	tests := []struct {
		name      string
		allow     bool
		allowlist []string
		override  string
		status    int
		host      string
	}{
		{"no override", false, nil, "", 0, "cfg"},
		{"disabled", false, nil, "8.8.8.8", 403, ""},
		{"public ok", true, nil, "8.8.8.8", 0, "8.8.8.8"},
		{"public v6 ok", true, nil, "2001:4860:4860::8888", 0, "2001:4860:4860::8888"},
		{"loopback", true, nil, "127.0.0.1", 403, ""},
		{"private", true, nil, "10.0.0.1", 403, ""},
		{"link-local", true, nil, "169.254.169.254", 403, ""},
		{"v4-mapped loopback", true, nil, "::ffff:127.0.0.1", 403, ""},
		{"unspecified", true, nil, "0.0.0.0", 403, ""},
		{"cgnat", true, nil, "100.64.0.1", 403, ""},
		{"benchmarking", true, nil, "198.18.0.1", 403, ""},
		{"documentation", true, nil, "203.0.113.5", 403, ""},
		{"reserved", true, nil, "240.0.0.1", 403, ""},
		{"nat64", true, nil, "64:ff9b::a00:1", 403, ""},
		{"6to4", true, nil, "2002:a00:1::1", 403, ""},
		{"teredo", true, nil, "2001:0:4136:e378::1", 403, ""},
		{"ipv6 doc", true, nil, "2001:db8::1", 403, ""},
		{"just outside cgnat", true, nil, "100.128.0.1", 0, "100.128.0.1"},
		{"cgnat allowlisted", true, []string{"100.64.0.0/10"}, "100.64.0.1", 0, "100.64.0.1"},
		{"hostname", true, nil, "example.com", 400, ""},
		{"zone", true, nil, "fe80::1%eth0", 400, ""},
		{"allowlist cidr", true, []string{"10.0.0.0/8"}, "10.1.2.3", 0, "10.1.2.3"},
		{"allowlist ip", true, []string{"1.1.1.1"}, "1.1.1.1", 0, "1.1.1.1"},
		{"allowlist miss", true, []string{"1.1.1.1"}, "8.8.8.8", 403, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config = &Config{DNSServer: "cfg", AllowDNSServer: tc.allow, DNSServerAllowlist: tc.allowlist}
			host, status, _ := pickUpstream(tc.override)
			if status != tc.status || host != tc.host {
				t.Errorf("got host=%q status=%d", host, status)
			}
		})
	}
}

func TestDNSServerOverrideHTTP(t *testing.T) {
	startUpstream(t)
	good := config.DNSServer
	config.DNSServer = "192.0.2.1" // dead; success proves the override was used
	config.AllowDNSServer = true
	config.DNSServerAllowlist = []string{good}
	code, got := query(t, "name=ok.test&type=1&dnsserver="+good)
	if code != 200 || len(got.Answer) != 1 {
		t.Errorf("override: code=%d answers=%d", code, len(got.Answer))
	}
	code, got = query(t, "name=ok.test&type=1&dnsserver=8.8.8.8")
	if code != 403 || got.Comment == "" {
		t.Errorf("not allowlisted: code=%d comment=%q", code, got.Comment)
	}
}
