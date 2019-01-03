package entrypoint

import (
	"fmt"
	. "github.com/k3rn3l-p4n1c/apigateway"
)

type Server interface {
	Start() error
	Close() error
	EqualConfig(c *EntryPoint) bool
}

func New(config *EntryPoint, handle HandleFunc) (Server, error) {
	switch config.Protocol {
	case "http":
		if config.Enabled != nil && !*config.Enabled {
			return nil, fmt.Errorf("%s server is not enabled in config", config.Protocol)
		}
		return &Http{
			config:  config,
			handle:  handle,
		}, nil

	default:
		return nil, fmt.Errorf("protocol %s for frontend is not supported", config.Protocol)
	}
}
