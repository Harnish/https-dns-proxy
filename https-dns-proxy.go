package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ajays20078/go-http-logger"
	"github.com/miekg/dns"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strconv"
	"time"
)

type QuestionRec struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}

type ResponseRecord struct {
	AD               bool          `json:"AD"`
	Additional       []interface{} `json:"Additional"`
	Answer           []dns.RR
	CD               bool        `json:"CD"`
	Question         QuestionRec `json:"Question"`
	RA               bool        `json:"RA"`
	RD               bool        `json:"RD"`
	Status           int         `json:"Status"`
	TC               bool        `json:"TC"`
	EdnsClientSubnet string      `json:"edns_client_subnet"`
	Comment          string      `json:"Comment"`
}

var port = flag.String("port", "8414", "Port you want to listen on")
var dnsserver = flag.String("dnsserver", "8.8.8.8", "DNS server you want to use as your source")
var dnsport = flag.String("dnsport", "53", "Port on the DNS server to talk to")
var sslkeypath = flag.String("sslkeypath", "", "Path to SSL Key file")
var sslcrtpath = flag.String("sslcrtpath", "", "Path to SSL CRT file")
var loglocation = flag.String("log", "", "Directory for log file.  Will not log if param is missing")
var allowdnsserver = flag.Bool("allow-dnsserver", false, "Let clients pick the upstream DNS server with ?dnsserver=<ip>. Public IPs only unless dnsserverallowlist is set in the config file")
var configfilelocation = flag.String("conf", "", "Location of a config file.  Will override passed in parameters")

func writeJSON(w http.ResponseWriter, code int, rec ResponseRecord) {
	b, err := json.Marshal(rec)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	w.Write(b)
}

// pickUpstream returns the DNS server IP/host to query. override is the
// client's ?dnsserver= value; it is ignored unless AllowDNSServer is set.
// With an allowlist (IPs or CIDRs) only listed addresses are accepted;
// without one only public unicast IPs are, so callers can't aim the server
// at loopback, private, or link-local hosts.
func pickUpstream(override string) (host string, status int, msg string) {
	if override == "" {
		return config.DNSServer, 0, ""
	}
	if !config.AllowDNSServer {
		return "", http.StatusForbidden, "dnsserver override disabled"
	}
	addr, err := netip.ParseAddr(override)
	if err != nil || addr.Zone() != "" {
		return "", http.StatusBadRequest, "invalid dnsserver (IP address required)"
	}
	addr = addr.Unmap()
	if len(config.DNSServerAllowlist) > 0 {
		for _, e := range config.DNSServerAllowlist {
			if p, err := netip.ParsePrefix(e); err == nil && p.Contains(addr) {
				return addr.String(), 0, ""
			}
			if a, err := netip.ParseAddr(e); err == nil && a.Unmap() == addr {
				return addr.String(), 0, ""
			}
		}
		return "", http.StatusForbidden, "dnsserver not in allowlist"
	}
	if !addr.IsGlobalUnicast() || addr.IsPrivate() {
		return "", http.StatusForbidden, "dnsserver must be a public IP"
	}
	return addr.String(), 0, ""
}

func ResolveDNS(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	q := req.URL.Query()
	recname := q.Get("name")
	rectype := q.Get("type")
	dnssec := q.Get("dnssec") == "1" || q.Get("dnssec") == "true"
	cd := q.Get("cd") == "1" || q.Get("cd") == "true"

	rectypeint := 255
	fail := func(code, rcode int, comment string) {
		writeJSON(w, code, ResponseRecord{
			Question: QuestionRec{Name: recname, Type: rectypeint},
			Status:   rcode,
			Comment:  comment,
		})
	}
	if recname == "" {
		fail(http.StatusBadRequest, dns.RcodeFormatError, "missing name")
		return
	}
	if rectype != "" {
		t, err := strconv.ParseUint(rectype, 10, 16)
		if err != nil {
			fail(http.StatusBadRequest, dns.RcodeFormatError, "invalid type")
			return
		}
		rectypeint = int(t)
	}

	host, code, msg := pickUpstream(q.Get("dnsserver"))
	if code != 0 {
		fail(code, dns.RcodeRefused, msg)
		return
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(recname), uint16(rectypeint))
	m.RecursionDesired = true
	m.CheckingDisabled = cd
	if dnssec {
		m.SetEdns0(4096, true)
	}
	upstream := net.JoinHostPort(host, config.DNSPort)
	c := new(dns.Client)
	r, _, err := c.Exchange(m, upstream)
	if r != nil && r.Truncated {
		c.Net = "tcp"
		r, _, err = c.Exchange(m, upstream)
	}
	if r == nil {
		log.Printf("upstream error for %q: %v", recname, err)
		fail(http.StatusBadGateway, dns.RcodeServerFailure, "upstream DNS error")
		return
	}

	comment := ""
	if r.Rcode != dns.RcodeSuccess {
		comment = dns.RcodeToString[r.Rcode]
	}
	writeJSON(w, http.StatusOK, ResponseRecord{
		AD:       r.AuthenticatedData,
		Answer:   r.Answer,
		CD:       r.CheckingDisabled,
		Question: QuestionRec{Name: recname, Type: rectypeint},
		RA:       r.RecursionAvailable,
		RD:       r.RecursionDesired,
		Status:   r.Rcode,
		TC:       r.Truncated,
		Comment:  comment,
	})
}

func ResolveDNSHTML(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, PageHTML)
}

func redirect(w http.ResponseWriter, r *http.Request) {

	http.Redirect(w, r, "/query", http.StatusFound)
}

var config *Config

func main() {
	flag.Parse()
	config = LoadConfig(*configfilelocation)
	if config.ListenPort == "" {
		config.ListenPort = *port
	}
	if config.SSLCrtPath == "" {
		config.SSLCrtPath = *sslcrtpath
	}
	if config.SSLKeyPath == "" {
		config.SSLKeyPath = *sslkeypath
	}
	if config.DNSServer == "" {
		config.DNSServer = *dnsserver
	}
	if config.DNSPort == "" {
		config.DNSPort = *dnsport
	}
	if !config.AllowDNSServer {
		config.AllowDNSServer = *allowdnsserver
	}
	for _, e := range config.DNSServerAllowlist {
		_, perr := netip.ParsePrefix(e)
		_, aerr := netip.ParseAddr(e)
		if perr != nil && aerr != nil {
			log.Fatalf("invalid dnsserverallowlist entry %q", e)
		}
	}
	if config.LogPath == "" {
		config.LogPath = *loglocation
	}

	var handler http.Handler = http.DefaultServeMux
	if config.LogPath != "" {
		f, err := os.OpenFile(config.LogPath+"/dns-access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Println("access log disabled:", err)
		} else {
			handler = httpLogger.WriteLog(handler, f)
		}
	}
	http.HandleFunc("/", redirect)
	http.HandleFunc("/query", ResolveDNSHTML)
	http.HandleFunc("/resolve", ResolveDNS)
	fmt.Printf("Starting webserver: %+v\n", config)

	srv := &http.Server{
		Addr:              ":" + config.ListenPort,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if config.SSLKeyPath != "" {
		log.Fatal("ListenAndServeTLS: ", srv.ListenAndServeTLS(config.SSLCrtPath, config.SSLKeyPath))
	}
	log.Fatal("ListenAndServe: ", srv.ListenAndServe())
}
