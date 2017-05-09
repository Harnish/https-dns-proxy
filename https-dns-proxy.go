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
	rectypeint, err := strconv.Atoi(rectype)
	if err != nil {
		rectypeint = 255
	}
	myquestion := QuestionRec{
		Name: recname,
		Type: rectypeint,
	}
	m.SetQuestion(dns.Fqdn(recname), uint16(rectypeint))
	m.RecursionDesired = true
	r, _, err := c.Exchange(m, net.JoinHostPort(config.DNSServer, config.DNSPort))
	if r == nil {
		log.Fatalf("*** error: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	status := 0
	if r.Rcode != dns.RcodeSuccess {
		log.Fatalf(" *** invalid answer name %s after A query for %s\n", recname, recname)
		status = 1

	}

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

	http.Redirect(w, r, "/query", 301)
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

	access_file_handler, err := os.OpenFile(config.LogPath+"/dns-access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err)
		config.LogPath = ""
	}
	http.HandleFunc("/", redirect)
	http.HandleFunc("/query", ResolveDNSHTML)
	http.HandleFunc("/resolve", ResolveDNS)
	fmt.Printf("Starting webserver: %+v\n", config)
	if config.LogPath != "" {
		if config.SSLKeyPath != "" {
			err := http.ListenAndServeTLS(":"+config.ListenPort, config.SSLCrtPath, config.SSLKeyPath, httpLogger.WriteLog(http.DefaultServeMux, access_file_handler))
			if err != nil {
				log.Fatal("ListenAndServeTLS: ", err)
			}
		} else {
			err := http.ListenAndServe(":"+config.ListenPort, httpLogger.WriteLog(http.DefaultServeMux, access_file_handler))

			if err != nil {
				log.Fatal("ListenAndServe: ", err)
			}

		}
	} else {
		if config.SSLKeyPath != "" {
			err := http.ListenAndServeTLS(":"+config.ListenPort, config.SSLCrtPath, config.SSLKeyPath, nil)
			if err != nil {
				log.Fatal("ListenAndServeTLS: ", err)
			}
		} else {
			err := http.ListenAndServe(":"+config.ListenPort, nil)

			if err != nil {
				log.Fatal("ListenAndServe: ", err)
			}

		}
	}
}
