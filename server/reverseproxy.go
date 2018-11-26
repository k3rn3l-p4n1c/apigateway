package server

import (
	"net/http/httputil"
	"net/http"
	"net/url"
	"github.com/sirupsen/logrus"
)

func (h *Http) getReverseProxy(r *http.Request) (*httputil.ReverseProxy, error) {
	service, err := h.discovery.GetService(r)
	if err != nil {
		return nil, err
	}
	if proxy, ok := h.proxies[service.Url]; ok {
		return proxy, nil
	} else {
		logrus.Debugf("setting up reverse proxy to service %s", service.Name)
		serviceUrl, err := url.Parse(service.Url)
		if err != nil {
			return nil, err
		}
		proxy = httputil.NewSingleHostReverseProxy(serviceUrl)
		h.proxies[service.Url] = proxy
		return proxy, nil
	}
}
