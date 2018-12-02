package reproxy

import (
	"github.com/k3rn3l-p4n1c/apigateway"
	"github.com/sirupsen/logrus"
)

type Interface interface {
	Serve(Request apigateway.Request, backend *apigateway.Backend)
}

type MuxReverseProxy struct {
	httpReverseProxy *HttpReverseProxy
}

func NewReverseProxy(discovery *ServiceDiscovery) (Interface, error) {
	return &MuxReverseProxy{
		httpReverseProxy: NewHttpReverseProxy(discovery),
	}, nil
}

func (p MuxReverseProxy) Serve(request apigateway.Request, backend *apigateway.Backend) {
	switch request.Protocol {
	case "http":
		p.httpReverseProxy.Server(request, backend)
	default:
		logrus.Errorf("invalid protocol %s", request.Protocol)
		request.HttpResponseWriter.Write([]byte("protocol is wrong"))
	}
}
