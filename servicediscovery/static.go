package servicediscovery

import (
	"github.com/k3rn3l-p4n1c/apigateway"
	"net/url"
)

type StaticServiceDiscovery struct {
	config apigateway.Discovery
}

func (discovery *StaticServiceDiscovery) Get(request *apigateway.Request) (scheme string, host string, path string, err error) {
	backendUrl, err := url.Parse(discovery.config.Url)
	println(backendUrl.Scheme, backendUrl.Host, backendUrl.Path)
	return backendUrl.Scheme, backendUrl.Host, backendUrl.Path, err

}
