package servicediscovery

import (
	"github.com/k3rn3l-p4n1c/apigateway/configuration"
	"net/http"
	"errors"
)

type ServiceDiscovery struct {
	config *configuration.Config
}

func MatchConditions(request *http.Request, frontend *configuration.Frontend) bool {
	return frontend.Host == request.Host
}

func (sd *ServiceDiscovery) GetBackend(request *http.Request) (*configuration.Backend, error) {
	for _, frontend := range sd.config.Frontend {
		if MatchConditions(request, frontend) {
			return frontend.ToBackend, nil
		}
	}
	return nil, errors.New("not found")
}

func NewServiceDiscovery(config *configuration.Config) (*ServiceDiscovery, error) {
	return &ServiceDiscovery{
		config: config,
	}, nil
}
