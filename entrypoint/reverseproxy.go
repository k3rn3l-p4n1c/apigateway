package entrypoint

import (
	"net/http/httputil"
	"net/http"
	"net/url"
	"github.com/sirupsen/logrus"
)

func (h *Http) getReverseProxy(r *http.Request) (*httputil.ReverseProxy, error) {
	backend, err := h.discovery.GetBackend(r)
	if err != nil {
		return nil, err
	}
	if proxy, ok := h.proxies[backend.Url]; ok {
		return proxy, nil
	} else {
		logrus.Debugf("setting up reverse proxy to service %s", backend.Name)
		serviceUrl, err := url.Parse(backend.Url)
		if err != nil {
			return nil, err
		}
		proxy = httputil.NewSingleHostReverseProxy(serviceUrl)
		h.proxies[backend.Url] = proxy
		return proxy, nil
	}
}
