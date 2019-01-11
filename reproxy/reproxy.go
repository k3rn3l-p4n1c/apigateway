package reproxy

import (
	"fmt"
	. "github.com/k3rn3l-p4n1c/apigateway"
	"github.com/k3rn3l-p4n1c/apigateway/servicediscovery"
)

func New(backend *Backend) (ReverseProxy, error) {
	discovery, err := servicediscovery.New(backend.Discovery)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize new reverse proxy. error=%v", err)
	}
	switch backend.Protocol {
	case "http":
		return NewHttpReverseProxy(discovery, backend)
	default:
		return nil, fmt.Errorf("invalid protocol %s", backend.Protocol)
	}
}
