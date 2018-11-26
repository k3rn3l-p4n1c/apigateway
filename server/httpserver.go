package server

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"github.com/k3rn3l-p4n1c/apigateway/configuration"
	"github.com/k3rn3l-p4n1c/apigateway/servicediscovery"
	"net/http/httputil"
)

type Http struct {
	config    *configuration.Server
	server    *http.Server
	discovery *servicediscovery.ServiceDiscovery
	proxies   map[string]*httputil.ReverseProxy
}

func (h *Http) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("called [%s] /%s", r.Method, r.URL.Path[1:])
	reproxy, err := h.getReverseProxy(r)
	if err != nil {
		logrus.WithError(err).Debugf("failed to get reverse proxy")
		fmt.Fprintf(w, "Error: %s!", err)
	} else {
		reproxy.ServeHTTP(w, r)
	}
}

func (h *Http) Start() error {
	logrus.Infof("start listening on %s", h.config.Addr)
	h.server = &http.Server{Addr: h.config.Addr, Handler: h}
	return h.server.ListenAndServe()
}

func (h *Http) Close() error {
	return h.server.Close()
}
