package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/oschwald/geoip2-golang"
	"gopkg.in/yaml.v2"
)

const (
	TCP_QUERY       = "tcp"
	UDP_QUERY       = "udp"
	POLICY_INCLUDED = "included"
	POLICY_EXCLUDED = "excluded"
)

type Configuration struct {
	Debug       bool
	Listen      string      `yaml:"listen"`
	ListenTCP   bool        `yaml:"listen_tcp"`
	ListenUDP   bool        `yaml:"listen_udp"`
	Resolvers   []*Resolver `yaml:"resolvers"`
	GeoIPDBPath string      `yaml:"geoip_db"`
	geoipDB     *geoip2.Reader
	cpuprofile  string
	memprofile  string
}

func defaultConfig() *Configuration {
	conf := new(Configuration)
	conf.Debug = false
	conf.Listen = "127.0.0.1:8053"
	conf.ListenTCP = false
	conf.ListenUDP = true
	conf.GeoIPDBPath = ""
	// TODO: default config
	return conf
}

var conf = defaultConfig()
var initialized bool

func GetConfig() (*Configuration, error) {
	if initialized {
		return conf, nil
	}
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile := flag.String("memprofile", "", "write memory profile to this file")
	configPath := flag.String("config", "dnsgo.yml", "load config")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()

	if *cpuprofile != "" {
		conf.cpuprofile = *cpuprofile
	}
	if *memprofile != "" {
		conf.memprofile = *memprofile
	}

	buffer, err := ioutil.ReadFile(*configPath)
	if err != nil {
		return conf, err
	}
	if err := yaml.Unmarshal(buffer, conf); err != nil {
		return conf, err
	}
	if conf.Debug {
		s, _ := yaml.Marshal(conf)
		fmt.Println("-----------config-start-----------")
		fmt.Fprintln(os.Stderr, string(s))
		fmt.Println("-----------config-stop-----------")
	}
	// TODO: validate config
	if conf.Debug {
		fmt.Println("opening geoipmmdb.")
	}
	conf.geoipDB, err = geoip2.Open(conf.GeoIPDBPath)
	if err != nil {
		return conf, err
	}

	for _, servers := range conf.Resolvers {
		for i, server := range servers.Server {
			if !strings.Contains(server, ":") {
				servers.Server[i] = server + ":53"
			}
		}

	}
	initialized = true
	return conf, nil
}
