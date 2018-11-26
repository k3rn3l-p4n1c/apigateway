package server

import (
	"fmt"
	"errors"
	"github.com/k3rn3l-p4n1c/apigateway/configuration"
	"github.com/k3rn3l-p4n1c/apigateway/servicediscovery"
	"net/http/httputil"
)

type Server interface {
	Start() error
	Close() error
}

func Factory(config *configuration.Server, sd *servicediscovery.ServiceDiscovery) (Server, error) {
	switch config.Protocol {
	case "http":
		return &Http{
			config:    config,
			discovery: sd,
			proxies:   make(map[string]*httputil.ReverseProxy),
		}, nil

	default:
		return nil, errors.New(fmt.Sprintf("protocol %s for frontend is not supported", config.Protocol))
	}
}
