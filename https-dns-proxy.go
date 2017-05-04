package main

import (
	"encoding/json"
	"fmt"
	"github.com/miekg/dns"
	"log"
	"net"
	"net/http"
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

var config = LoadConfig("/etc/dnsproxy.yaml")

func main() {

	http.HandleFunc("/", redirect)
	http.HandleFunc("/query", ResolveDNSHTML)
	http.HandleFunc("/resolve", ResolveDNS)
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
