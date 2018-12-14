package entrypoint

import (
	"fmt"
	"github.com/k3rn3l-p4n1c/apigateway"
	"net/http/httputil"
)

type Server interface {
	Start() error
	Close() error
	EqualConfig(c *apigateway.EntryPoint) bool
}

func Factory(config *apigateway.EntryPoint, handle apigateway.HandleFunc) (Server, error) {
	switch config.Protocol {
	case "http":
		if config.Enabled != nil && !*config.Enabled {
			return nil, fmt.Errorf("%s server is not enabled in config", config.Protocol)
		}
		return &Http{
			config:  config,
			handle:  handle,
			proxies: make(map[string]*httputil.ReverseProxy),
		}, nil

	default:
		return nil, fmt.Errorf("protocol %s for frontend is not supported", config.Protocol)
	}
}
