package cache

type CustomDNSConfig struct {
	Prefix    string
	DNSServer string
}

type CustomeDNSConfigCache map[string]CustomDNSConfig

func NewCustomeDNSConfigCache() CustomeDNSConfigCache {
	cc := make(CustomeDNSConfigCache)
	return cc
}

func (cc CustomeDNSConfigCache) Get(prefix string) (CustomDNSConfig, bool) {
	c, existed := cc[prefix]
	return c, existed
}

func (cc CustomeDNSConfigCache) Set(c CustomDNSConfig) {
	cc[c.Prefix] = c
}

// func (cc CustomeDNSConfigCache) BuildCacheFromFile(file *os.File) error {

// }
