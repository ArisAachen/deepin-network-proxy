package DBus

import (
	"net"

	"github.com/golang/groupcache/lru"
)

type fakeIPCache struct {
	cache *lru.Cache

	mapper map[uint32]string
}

func newFakeIPCache() *fakeIPCache {
	f := new(fakeIPCache)
	f.cache = lru.New(cacheMaxSize)
	f.mapper = make(map[uint32]string)

	return f
}

func (f *fakeIPCache) Add(domain string, ip net.IP) {
	f.cache.Add(domain, ip)

	ipUint := ipToUint(ip)
	f.mapper[ipUint] = domain
}

func (f *fakeIPCache) GetByDomain(domain string) (net.IP, bool) {
	ipI, ok := f.cache.Get(domain)
	if !ok {
		return net.IP{}, false
	}

	return ipI.(net.IP), true
}

func (f *fakeIPCache) GetByIP(ip net.IP) (domain string, ok bool) {
	ipUint := ipToUint(ip)

	domain, ok = f.mapper[ipUint]
	if !ok {
		return
	}

	_, ok = f.cache.Get(domain)
	if !ok {
		delete(f.mapper, ipUint)
		return
	}

	return
}
