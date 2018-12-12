package servicediscovery

import (
	"fmt"
	"github.com/k3rn3l-p4n1c/apigateway"
)

func New(config apigateway.Discovery) (apigateway.ServiceDiscovery, error) {
	switch config.Type {
	case "static":
		return &StaticServiceDiscovery{
			config: config,
		}, nil
	default:
		return nil, fmt.Errorf("discovery type %s is not supported", config.Type)

	}
}
