package dns

import (
	"fmt"
	"net"
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
	DNSClient             *dns.Client
	DefaultDNSServer      string
}

func NewAgent() *Agent {
	agent := &Agent{}
	agent.Initialize()
	return agent
}

// Initialize a DNS Agent
func (a *Agent) Initialize() {
	a.ACache = cache.NewDomainCache()
	a.AAAACache = cache.NewDomainCache()
	a.CustomeDNSConfigCache = cache.NewCustomeDNSConfigCache()
	a.DNSClient = new(dns.Client)
	a.DNSClient.Net = "udp"

	a.CustomeDNSConfigCache.Set(cache.CustomDNSConfig{
		Prefix:    "baidu.com.",
		DNSServer: "119.29.29.29",
	})
	a.CustomeDNSConfigCache.Set(cache.CustomDNSConfig{
		Prefix:    "google.com.",
		DNSServer: "1.1.1.1",
	})
}

// ServeDNS make a DNS query response
func (a *Agent) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)

	domain := msg.Question[0].Name
	answer, err := a.LookupDNS(r.Question[0].Qtype, domain)

	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true

		glog.Infof("Query: %s %s", r.Question[0].Qtype, domain)
		// address, ok := domainsToAddresses[domain]
		// _, dnsServer := a.QueryDNSServer(r.Question[0].Qtype, domain)

		if err == nil {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   domain,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    60,
				},
				A: net.ParseIP(answer),
			})
		}
	}
	w.WriteMsg(&msg)
}

// QueryDNSServer get DNS results
func (a *Agent) LookupDNS(qType uint16, name string) (string, error) {
	glog.Info(a.ACache)
	switch qType {
	case dns.TypeA:
		c, exists := a.ACache.Get(name)
		// Cache hit, return cache
		if exists {
			return c.Record, nil
		}
	}

	// Cache missed, check in Spec List
	dnsServer := a.GetDNSServer(name)
	glog.Infof("Query [%s]%s from DNS server %s", qType, name, dnsServer)

	m := new(dns.Msg)
	m.SetQuestion(name, qType)
	m.RecursionDesired = true
	r, _, err := a.DNSClient.Exchange(m, dnsServer)
	if err != nil {
		return "", err
	}
	if r.Rcode != dns.RcodeSuccess {
		return "", fmt.Errorf("Rcode=%d not successful", r.Rcode)
	}

	var answers []string
	for _, answer := range r.Answer {
		if field, ok := answer.(*dns.A); ok {
			glog.Infof("%s Answer: %s", name, field.A.String())
			answers = append(answers, field.A.String())
		}
	}

	switch qType {
	case dns.TypeA:
		a.ACache.Put(cache.Cache{
			Name:      name,
			Record:    answers[0],
			ExpiredAt: time.Now().Add(time.Duration(10) * time.Second),
		})
	}

	return answers[0], nil
}

// GetDNSServer check domain in custom set
func (a *Agent) GetDNSServer(name string) string {
	splitedName := strings.Split(name, ".")
	splitedLength := len(splitedName)
	for i := 0; i < splitedLength; i++ {
		glog.Infof("Find %s", splitedName[i:splitedLength])
		tmpName := strings.Join(splitedName[i:splitedLength], ".")
		c, existed := a.CustomeDNSConfigCache.Get(tmpName)
		if existed {
			return appendDefaultPort(c.DNSServer, "53")
		}
	}
	// Default result
	return a.DefaultDNSServer
}

// //
// func (a *Agent) LookupDNS(qType uint16, name, dnsServer string) (string, error) {
// 	glog.Infof("Lookup DNS %d %s @%s", qType, name, dnsServer)
// 	dnsClient := new(dns.Client)

// 	// TODO: more protoctols
// 	// - udp, tcp, DoH
// 	// - udp4, udp6, tcp4, tcp6, udp, tcp
// 	dnsClient.Net = "udp"

// 	m := new(dns.Msg)
// 	m.SetQuestion(name, qType)
// 	m.RecursionDesired = true
// 	r, _, err := dnsClient.Exchange(m, dnsServer)
// 	if err != nil {
// 		return "", err
// 	}
// 	if r.Rcode != dns.RcodeSuccess {
// 		return "", fmt.Errorf("Rcode=%d not successful", r.Rcode)
// 	}

// 	var answers []string
// 	for _, answer := range r.Answer {
// 		if field, ok := answer.(*dns.A); ok {
// 			glog.Infof("%s Answer: %s", name, field.A.String())
// 			answers = append(answers, field.A.String())
// 		}
// 	}

// 	return answers[0], nil
// }

func appendDefaultPort(serverIP, defaultPort string) string {
	idx := strings.Index(serverIP, ":")
	if idx > 1 {
		return serverIP
	}

	return serverIP + ":" + defaultPort
}
