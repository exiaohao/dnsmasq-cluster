package dns

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/exiaohao/home-dns/pkg/cache"
	"github.com/golang/glog"
	"github.com/miekg/dns"
)

type Agent struct {
	ACache                cache.DomainCache
	AAAACache             cache.DomainCache
	CustomeDNSConfigCache cache.CustomeDNSConfigCache
	CacheTTL              uint16
	DNSClient             *dns.Client
	DefaultDNSServer      string
}

// NewAgent creates a new agent
func NewAgent(c ServerConfig) *Agent {
	agent := &Agent{}
	agent.Initialize(c)
	return agent
}

// Initialize a DNS Agent
func (a *Agent) Initialize(c ServerConfig) {
	a.ACache = cache.NewDomainCache()
	a.AAAACache = cache.NewDomainCache()
	a.CustomeDNSConfigCache = cache.NewCustomeDNSConfigCache()
	a.DNSClient = new(dns.Client)
	a.DNSClient.Net = "udp"
	a.DefaultDNSServer = c.BaseDNS
	a.CacheTTL = c.CacheExpire

	// Load customDNSConfigs
	for _, f := range c.CustomResolveFiles {
		for _, c := range parseCustomResolveFileToCache(f) {
			a.CustomeDNSConfigCache.Set(c)
		}
	}
}

// ServeDNS make a DNS query response
func (a *Agent) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	msg.Authoritative = true

	domain := msg.Question[0].Name
	answers, err := a.LookupDNS(r.Question[0].Qtype, domain)
	if err != nil {
		w.WriteMsg(&msg)
	}

	switch r.Question[0].Qtype {
	case dns.TypeA:
		for _, answer := range answers {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   domain,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    uint32(a.CacheTTL),
				},
				A: net.ParseIP(answer.Result),
			})
		}
	case dns.TypeCNAME:
		for _, answer := range answers {
			msg.Answer = append(msg.Answer, &dns.CNAME{
				Hdr: dns.RR_Header{
					Name:   domain,
					Rrtype: dns.TypeCNAME,
					Class:  dns.ClassINET,
					Ttl:    uint32(a.CacheTTL),
				},
				Target: answer.Result,
			})
		}
	case dns.TypeAAAA:
		for _, answer := range answers {
			msg.Answer = append(msg.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   domain,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    uint32(a.CacheTTL),
				},
				AAAA: net.ParseIP(answer.Result),
			})
		}
	case dns.TypeMX:
		for _, answer := range answers {
			msg.Answer = append(msg.Answer, &dns.MX{
				Hdr: dns.RR_Header{
					Name:   domain,
					Rrtype: dns.TypeMX,
					Class:  dns.ClassINET,
					Ttl:    uint32(a.CacheTTL),
				},
				Mx:         answer.Result,
				Preference: answer.Preference,
			})
		}
	}
	w.WriteMsg(&msg)
}

// QueryDNSServer get DNS results
func (a *Agent) LookupDNS(qType uint16, name string) ([]cache.Answer, error) {
	switch qType {
	case dns.TypeA:
		c, exists := a.ACache.Get(name)
		// Cache hit, return cache
		if exists {
			return c.Records, nil
		}
	case dns.TypeAAAA:
		c, exists := a.AAAACache.Get(name)
		if exists {
			return c.Records, nil
		}
	}

	// Cache missed, check in Spec List
	dnsServer := a.GetDNSServer(name)

	m := new(dns.Msg)
	m.SetQuestion(name, qType)
	m.RecursionDesired = true
	r, _, err := a.DNSClient.Exchange(m, dnsServer)
	if err != nil {
		return []cache.Answer{}, err
	}
	if r.Rcode != dns.RcodeSuccess {
		return []cache.Answer{}, fmt.Errorf("Rcode=%d not successful", r.Rcode)
	}

	var answers []cache.Answer
	for _, answer := range r.Answer {
		fieldA, ok := answer.(*dns.A)
		if ok {
			answers = append(answers, cache.Answer{
				Result:     fieldA.A.String(),
				Preference: 0,
			})
			continue
		}
		fieldAAAA, ok := answer.(*dns.AAAA)
		if ok {
			answers = append(answers, cache.Answer{
				Result:     fieldAAAA.AAAA.String(),
				Preference: 0,
			})
			continue
		}
		fieldCNAME, ok := answer.(*dns.CNAME)
		if ok {
			answers = append(answers, cache.Answer{
				Result:     fieldCNAME.Target,
				Preference: 0,
			})
			continue
		}
		fieldMX, ok := answer.(*dns.MX)
		if ok {
			answers = append(answers, cache.Answer{
				Result:     fieldMX.Mx,
				Preference: fieldMX.Preference,
			})
			glog.Info(fieldMX.Preference, fieldMX.Mx)
			continue
		}
	}

	switch qType {
	case dns.TypeA:
		a.ACache.Put(cache.Cache{
			Name:      name,
			Records:   answers,
			ExpiredAt: time.Now().Add(time.Duration(a.CacheTTL) * time.Second),
		})
	case dns.TypeAAAA:
		a.AAAACache.Put(cache.Cache{
			Name:      name,
			Records:   answers,
			ExpiredAt: time.Now().Add(time.Duration(a.CacheTTL) * time.Second),
		})
	}

	glog.Infof("Query [%d]%s from DNS server %s, Results: %s", qType, name, dnsServer, answers)
	return answers, nil
}

// GetDNSServer check domain in custom set
func (a *Agent) GetDNSServer(name string) string {
	splitedName := strings.Split(name, ".")
	splitedLength := len(splitedName)
	for i := 0; i < splitedLength-1; i++ {
		tmpName := strings.Join(splitedName[i:splitedLength], ".")
		c, existed := a.CustomeDNSConfigCache.Get(tmpName)
		if existed {
			glog.Infof("Match prefix `%s` from CustomeDNSConfig: %s Query DNS: %s", c.Prefix, name, c.DNSServer)
			return appendDefaultPort(c.DNSServer, "53")
		}
	}
	// Default result
	return a.DefaultDNSServer
}

func appendDefaultPort(serverIP, defaultPort string) string {
	idx := strings.Index(serverIP, ":")
	if idx > 1 {
		return serverIP
	}

	return serverIP + ":" + defaultPort
}

func Readln(r *bufio.Reader) (string, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

// parseCustomResolveFileToCache
// TODO: read file better
func parseCustomResolveFileToCache(file string) []cache.CustomDNSConfig {
	var customDNSConfigs []cache.CustomDNSConfig

	f, err := os.Open(file)
	if err != nil {
		glog.Errorf("Failed to open %s: %s", file, err)
		return customDNSConfigs
	}
	r := bufio.NewReader(f)
	s, err := Readln(r)
	for err == nil {
		// Skip comments
		skipFlag := strings.Index(s, "#")
		if skipFlag == 0 {
			glog.Info("Skip ", s)
			s, err = Readln(r)
			continue
		}
		// Add DOH Flag
		dohFlag := strings.Index(s, "#DOH")
		isDoHPerferred := false
		if dohFlag > 1 {
			isDoHPerferred = true
		}
		// Split config
		fields := strings.Split(s, "/")
		if len(fields) != 3 || fields[0] != "server=" {
			glog.Warningf("Bad config %s, skipped", s)
			continue
		}
		// Append . at last
		if fields[1][len(fields[1]):] != "." {
			fields[1] = fields[1] + "."
		}

		customDNSConfigs = append(customDNSConfigs, cache.CustomDNSConfig{
			Prefix:       fields[1],
			DNSServer:    fields[2],
			DoHPerferred: isDoHPerferred,
		})
		s, err = Readln(r)
	}
	return customDNSConfigs
}

func randomResult(answers []cache.Answer) cache.Answer {
	answersLength := len(answers)
	if answersLength == 1 {
		return answers[0]
	}
	return answers[rand.Intn(answersLength)]
}
