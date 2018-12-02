package servicediscovery

import (
	"net/url"
	"errors"
	"fmt"
	"github.com/k3rn3l-p4n1c/apigateway"
)

type ServiceDiscovery struct {
	config *apigateway.Config
}

func NewServiceDiscovery(config *apigateway.Config) *ServiceDiscovery {
	return &ServiceDiscovery{
		config: config,
	}
}

func (discovery *ServiceDiscovery) Get(backend *apigateway.Backend) (scheme string, host string, path string, err error) {
	switch backend.DiscoveryType {
	case "static":
		backendUrl, err := url.Parse(backend.Url)
		println(backendUrl.Scheme, backendUrl.Host, backendUrl.Path)
		return backendUrl.Scheme, backendUrl.Host, backendUrl.Path, err
	default:
		return "", "", "", errors.New(fmt.Sprintf("discovery type %s is not supported", backend.DiscoveryType))
	}
}
