package servicediscovery

import (
	"fmt"
	. "github.com/k3rn3l-p4n1c/apigateway"
)

func New(config Discovery) (ServiceDiscovery, error) {
	switch config.Type {
	case "static":
		d := &StaticServiceDiscovery{}
		err := d.setConfig(config)
		if err != nil {
			return nil, err
		}
		return d, nil
	case "dns":
		d := &DNSServiceDiscovery{}
		err := d.setConfig(config)
		if err != nil {
			return nil, err
		}
		return d, nil
	default:
		return nil, fmt.Errorf("discovery type %s is not supported", config.Type)

	}
}
