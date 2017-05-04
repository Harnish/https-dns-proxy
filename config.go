package main

import (
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"log"
	"os"
)

type Config struct {
	DNSServer  string
	DNSPort    string
	ListenPort string
	SSLKeyPath string
	SSLCrtPath string
}

func LoadConfig(path string) *Config {

	temp := new(Config)
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println("open config: ", err)
		os.Exit(1)
	}

	if err = yaml.Unmarshal(file, temp); err != nil {
		log.Println("parse config: ", err)
		os.Exit(1)
	}
	return temp
}
