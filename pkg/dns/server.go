package dns

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/golang/glog"
	"github.com/miekg/dns"
	"k8s.io/apimachinery/pkg/util/wait"
)

// InitOptions load configs on starts up
type InitOptions struct {
	BaseDNSServer string
	ConfigFile    string
}

type ServerConfig struct {
	BaseDNS            string   `yaml:"base_dns"`
	DOHServers         []string `yaml:"doh_servers"`
	WorkProtocol       string   `yaml:"work_protocol"`
	Listen             string   `yaml:"listen"`
	Port               string   `yaml:"port"`
	CacheExpire        uint16   `yaml:"cache_expire"`
	CustomResolveFiles []string `yaml:"custom_resolve_configs"`
}

func (sc *ServerConfig) serverListenPort() string {
	if sc.Port == "" {
		sc.Port = "53"
	}
	if sc.Listen == "" {
		sc.Listen = "127.0.0.1"
	}
	return sc.Listen + ":" + sc.Port
}

func (sc *ServerConfig) getWorkProtocol() string {
	if sc.WorkProtocol == "" {
		return "udp"
	}
	return sc.WorkProtocol
}

// Server with cache & server instance
type Server struct {
	Config      ServerConfig
	DNSServer   *dns.Server
	InitialFunc func()
}

// Initialize server
func (s *Server) Initialize(opts InitOptions) {
	glog.Info("Initialize DNS server...")
	// TODO: Load configfile
	configDat, err := ioutil.ReadFile(opts.ConfigFile)
	if err != nil {
		glog.Fatalf("Error reading config file, %s", err)
	}

	err = yaml.Unmarshal(configDat, &s.Config)
	if err != nil {
		glog.Fatalf("Failed to unmarshal config file: %s", err)
	}
	glog.Info("Load config successfully:", s.Config)

	s.DNSServer = &dns.Server{
		Addr: ":53", //s.Config.serverListenPort(),
		Net:  s.Config.getWorkProtocol(),
	}

	s.DNSServer.Handler = NewAgent(s.Config)
}

// Run a DNS server
func (s *Server) Run(stopCh <-chan struct{}) {
	glog.Info("Start running!")
	go wait.Until(s.RunDNSServerWorker, time.Microsecond, stopCh)

	<-stopCh
	glog.Info("Stopping server...")
}

// RunDNSServerWorker to process requests
func (s *Server) RunDNSServerWorker() {
	err := s.DNSServer.ListenAndServe()
	glog.Fatal(err)
}
