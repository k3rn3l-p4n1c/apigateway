package servicediscovery

import (
	. "github.com/k3rn3l-p4n1c/apigateway"
	"net"
	"fmt"
)

type StaticServiceDiscovery struct {
	config Discovery
	ip     string
}

func (discovery *StaticServiceDiscovery) Get(_ string) (ip string, err error) {
	return discovery.ip, nil
}

func (discovery *StaticServiceDiscovery) setConfig(config Discovery) (err error) {
	ip := net.ParseIP(discovery.config.Url)
	if ip == nil {
		return fmt.Errorf("invalid config: %s is not a valid IP", discovery.config.Url)
	} else {
		discovery.ip = ip.String()
		return nil
	}
}
