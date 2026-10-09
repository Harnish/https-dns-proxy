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
