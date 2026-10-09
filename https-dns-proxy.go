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
var configfilelocation = flag.String("conf", "", "Location of a config file.  Will override passed in parameters")

func ResolveDNS(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	recname := req.URL.Query().Get("name")
	rectype := req.URL.Query().Get("type")
	//FIXME add dnssec info
	//recdnssec := req.URL.Query().Get("dnssec")

	c := new(dns.Client)
	m := new(dns.Msg)
	rectypeint := 255
	if rectype != "" {
		t, err := strconv.ParseUint(rectype, 10, 16)
		if err != nil {
			http.Error(w, "invalid type", http.StatusBadRequest)
			return
		}
		rectypeint = int(t)
	}
	myquestion := QuestionRec{
		Name: recname,
		Type: rectypeint,
	}
	m.SetQuestion(dns.Fqdn(recname), uint16(rectypeint))
	m.RecursionDesired = true
	r, _, err := c.Exchange(m, net.JoinHostPort(config.DNSServer, config.DNSPort))
	if r == nil {
		log.Printf("upstream error for %s: %v", recname, err)
		http.Error(w, "upstream DNS error", http.StatusBadGateway)
		return
	}

	status := r.Rcode

	//FIXME make all fields updated
	responsejson := ResponseRecord{
		AD:       false,
		CD:       false,
		Answer:   r.Answer,
		Question: myquestion,
		Status:   status,
		TC:       false,
		RD:       true,
		RA:       true,
	}

	jsonoutbyte, err := json.Marshal(responsejson)
	if err != nil {
		fmt.Println("Error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write(jsonoutbyte)
	}
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
