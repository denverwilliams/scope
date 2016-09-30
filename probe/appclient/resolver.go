package appclient

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/miekg/dns"

	"github.com/weaveworks/scope/common/xfer"
)

const (
	dnsPollInterval = 10 * time.Second
)

var (
	tick = fastStartTicker
)

// fastStartTicker is a ticker that 'ramps up' from 1 sec to duration.
func fastStartTicker(duration time.Duration) <-chan time.Time {
	c := make(chan time.Time, 1)
	go func() {
		d := 1 * time.Second
		for {
			time.Sleep(d)
			d = d * 2
			if d > duration {
				d = duration
			}

			select {
			case c <- time.Now():
			default:
			}
		}
	}()
	return c
}

type setter func(string, []url.URL)

// Resolver is a thing that can be stopped...
type Resolver interface {
	Stop()
}

type staticResolver struct {
	setters           []setter
	targets           []target
	failedResolutions map[string]struct{}
	quit              chan struct{}
	lookup            LookupIP
}

// LookupIP type is used for looking up IPs.
type LookupIP func(host string) (ips []net.IP, err error)

type target struct {
	original string   // the original url string
	url      *url.URL // the parsed url
	hostname string   // the hostname (without port) from the url
	port     int      // the port, or a sensible default
}

func (t target) String() string {
	return net.JoinHostPort(t.hostname, strconv.Itoa(t.port))
}

// NewResolver periodically resolves the targets, and calls the set
// function with all the resolved IPs. It explictiy supports targets which
// resolve to multiple IPs.  It uses the supplied DNS server name.
func NewResolver(targets []string, lookup LookupIP, setters ...setter) (Resolver, error) {
	processed, err := prepare(targets)
	if err != nil {
		return nil, err
	}
	r := staticResolver{
		targets:           processed,
		setters:           setters,
		failedResolutions: map[string]struct{}{},
		quit:              make(chan struct{}),
		lookup:            lookup,
	}
	go r.loop()
	return r, nil
}

// LookupUsing produces a LookupIP function for the given DNS server.
func LookupUsing(dnsServer string) func(host string) (ips []net.IP, err error) {
	client := dns.Client{
		Net: "tcp",
	}
	return func(host string) (ips []net.IP, err error) {
		m := &dns.Msg{}
		m.SetQuestion(dns.Fqdn(host), dns.TypeA)
		in, _, err := client.Exchange(m, dnsServer)
		if err != nil {
			return nil, err
		}
		result := []net.IP{}
		for _, answer := range in.Answer {
			if a, ok := answer.(*dns.A); ok {
				result = append(result, a.A)
			}
		}
		return result, nil
	}
}

func (r staticResolver) loop() {
	r.resolve()
	t := tick(dnsPollInterval)
	for {
		select {
		case <-t:
			r.resolve()
		case <-r.quit:
			return
		}
	}
}

func (r staticResolver) Stop() {
	close(r.quit)
}

func prepare(urls []string) ([]target, error) {
	var targets []target
	for _, u := range urls {
		// naked hostnames (such as "localhost") are interpreted as relative URLs
		// so we add a scheme if u doesn't have one.
		if !strings.Contains(u, "//") {
			if strings.HasSuffix(u, ":443") {
				u = "https://" + u
			} else {
				u = "http://" + u
			}
		}
		parsed, err := url.Parse(u)
		if err != nil {
			return nil, err
		}

		var hostname string
		var port int
		if strings.Contains(parsed.Host, ":") {
			var portStr string
			hostname, portStr, err = net.SplitHostPort(parsed.Host)
			if err != nil {
				return nil, err
			}
			port, err = strconv.Atoi(portStr)
			if err != nil {
				return nil, err
			}
		} else {
			hostname, port = parsed.Host, xfer.AppPort
		}
		targets = append(targets, target{
			original: u,
			url:      parsed,
			hostname: hostname,
			port:     port,
		})
	}
	return targets, nil
}

func (r staticResolver) resolve() {
	for _, t := range r.targets {
		ips := r.resolveOne(t)
		urls := makeURLs(t, ips)
		for _, setter := range r.setters {
			setter(t.hostname, urls)
		}
	}
}

func makeURLs(t target, ips []string) []url.URL {
	result := []url.URL{}
	for _, ip := range ips {
		u := *t.url
		u.Host = net.JoinHostPort(ip, strconv.Itoa(t.port))
		result = append(result, u)
	}
	return result
}

func (r staticResolver) resolveOne(t target) []string {
	var addrs []net.IP
	if addr := net.ParseIP(t.hostname); addr != nil {
		addrs = []net.IP{addr}
	} else {
		var err error
		addrs, err = r.lookup(t.hostname)
		if err != nil {
			if _, ok := r.failedResolutions[t.hostname]; !ok {
				log.Warnf("Cannot resolve '%s': %v", t.hostname, err)
				// Only log the error once
				r.failedResolutions[t.hostname] = struct{}{}
			}
			return []string{}
		}
		// Allow logging errors in future resolutions
		delete(r.failedResolutions, t.hostname)
	}
	endpoints := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		// For now, ignore IPv6
		if addr.To4() == nil {
			continue
		}
		endpoints = append(endpoints, addr.String())
	}
	return endpoints
}
