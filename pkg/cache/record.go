package cache

import "time"

type Cache struct {
	Name      string
	Record    string
	ExpiredAt time.Time
}

// DomainCache build from queries
// For different types of record like A, AAAA, ... create different cache
// e.g.
// www.baidu.com -> {www.baidu.com A 115.239.210.27 ExpiredAt}
type DomainCache map[string]Cache

// NewDomainCache create a cache set
func NewDomainCache() DomainCache {
	c := make(DomainCache)
	return c
}

// Get record from DomainCache
func (dc DomainCache) Get(name string) (Cache, bool) {
	cache, existed := dc[name]
	if cache.ExpiredAt.Before(time.Now()) {
		delete(dc, name)
		return cache, false
	}
	return cache, existed
}

// Put record to DomainCache
func (dc DomainCache) Put(c Cache) {
	dc[c.Name] = c
}
