package servicediscovery

import (
	"github.com/k3rn3l-p4n1c/apigateway/configuration"
	"net/http"
	"errors"
)

type ServiceDiscovery struct {
	config *configuration.Config
}

func MatchConditions(request *http.Request, endpoint *configuration.Endpoint) bool {
	return endpoint.Host == request.Host
}

func (sd *ServiceDiscovery) GetService(request *http.Request) (*configuration.Service, error) {
	for _, endpoint := range sd.config.Endpoints {
		if MatchConditions(request, endpoint) {
			return endpoint.ToService, nil
		}
	}
	return nil, errors.New("not found")
}

func NewServiceDiscovery(config *configuration.Config) (*ServiceDiscovery, error) {
	return &ServiceDiscovery{
		config: config,
	}, nil
}
