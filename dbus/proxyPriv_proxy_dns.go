package DBus

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

const cacheMaxSize = 1000

type proxyDNS struct {
	prv *proxyPrv

	fIP   fakeIP
	cache *fakeIPCache
}

func (p *proxyDNS) resolveDomain(domain string) net.IP {
	i, ok := p.cache.GetByDomain(domain)
	if ok {
		return i
	}

	ip := p.fIP.new()
	logger.Debugf("Query for %s: %s", domain, ip)
	p.cache.Add(domain, ip)

	return ip
}

func (p *proxyDNS) getDomainFromFakeIP(ip net.IP) (string, bool) {
	return p.cache.GetByIP(ip)
}

func (p *proxyDNS) parseQuery(m *dns.Msg) {
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			ip := p.resolveDomain(q.Name)
			rr, err := dns.NewRR(fmt.Sprintf("%s 0 A %s", q.Name, ip))
			if err == nil {
				m.Answer = append(m.Answer, rr)
			}
		case dns.TypeAAAA:
			fmt.Printf("Query for %s\n", q.Name)
		}
	}
}

func (p *proxyDNS) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := &dns.Msg{}
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		p.parseQuery(m)
	}

	w.WriteMsg(m)
}

func (p *proxyDNS) startDNSProxy() error {
	p.fIP = newFakeIP(net.IP{225, 0, 0, 0}, 8)
	p.cache = newFakeIPCache()

	dnsListenAddr := fmt.Sprintf("127.0.0.1:%d", p.prv.Proxies.DNSPort)
	logger.Info("dns listen addr:", dnsListenAddr)

	server := &dns.Server{
		Addr:    dnsListenAddr,
		Net:     "udp",
		Handler: p,
	}

	return server.ListenAndServe()
}
