package servicediscovery

import (
	. "github.com/k3rn3l-p4n1c/apigateway"
	"net"
	"sync"
	"fmt"
	"time"
	"math/rand"
	"regexp"
	"github.com/sirupsen/logrus"
)

var domainRe = regexp.MustCompile(`[a-zA-Z0-9-_]+(\.[a-zA-Z0-9-_])*`)

type DNSServiceDiscovery struct {
	config     Discovery
	ips        []string
	lastUpdate time.Time
	mtx        sync.RWMutex
}

func (discovery *DNSServiceDiscovery) Get(_ string) (ip string, err error) {
	discovery.resolveDns()

	discovery.mtx.RLock()
	defer discovery.mtx.RUnlock()

	if len(discovery.ips) < 1 {
		return "", fmt.Errorf("no ip address discoverd for %s", discovery.config.Url)
	}
	ip = discovery.ips[rand.Intn(len(discovery.ips))]
	return ip, nil
}

func (discovery *DNSServiceDiscovery) setConfig(config Discovery) (err error) {
	if err := func() error {
		discovery.mtx.Lock()
		defer discovery.mtx.Unlock()

		if !domainRe.Match([]byte(config.Url)) {
			return fmt.Errorf("invalid url: %s is not a valid domain", config.Url)
		}
		discovery.config = config
		discovery.lastUpdate = time.Unix(0,0)
		return nil
	}(); err != nil {
		return err
	}

	discovery.resolveDns()
	return nil
}

func (discovery *DNSServiceDiscovery) resolveDns() {
	if time.Since(discovery.lastUpdate).Seconds() > 10 {
		discovery.mtx.Lock()
		defer discovery.mtx.Unlock()

		if ips, err := net.LookupIP(discovery.config.Url); err == nil {
			logrus.Debugf("resolve %d ip for %s", len(ips), discovery.config.Url)

			discovery.ips = make([]string, len(ips))
			for i, ip := range ips {
				discovery.ips[i] = ip.String()
			}
			discovery.lastUpdate = time.Now()
		} else {
			logrus.Debugf("unable to resolve ip for %s", discovery.config.Url)
		}
	}
}
