package entrypoint

import (
	"fmt"
	"errors"
	"net/http/httputil"
	"github.com/k3rn3l-p4n1c/apigateway"
)

type Server interface {
	Start() error
	Close() error
}

func Factory(config *apigateway.EntryPoint, handle apigateway.HandleFunc) (Server, error) {
	switch config.Protocol {
	case "http":
		if !*config.Enabled {
			return nil, errors.New(fmt.Sprintf("%s server is not enabled in config", config.Protocol))
		}
		return &Http{
			config:  config,
			handle:  handle,
			proxies: make(map[string]*httputil.ReverseProxy),
		}, nil

	default:
		return nil, errors.New(fmt.Sprintf("protocol %s for frontend is not supported", config.Protocol))
	}
}
