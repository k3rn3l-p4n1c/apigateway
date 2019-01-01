package servicediscovery

import (
	. "github.com/k3rn3l-p4n1c/apigateway"
	"net/url"
)

type StaticServiceDiscovery struct {
	config Discovery
}

func (discovery *StaticServiceDiscovery) Get(request *Request) (scheme string, host string, path string, err error) {
	backendUrl, err := url.Parse(discovery.config.Url)
	return backendUrl.Scheme, backendUrl.Host, backendUrl.Path, err
}
