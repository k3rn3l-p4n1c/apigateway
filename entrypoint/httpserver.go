package entrypoint

import (
	"context"
	"github.com/k3rn3l-p4n1c/apigateway"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

type Http struct {
	config  *apigateway.EntryPoint
	server  *http.Server
	proxies map[string]*httputil.ReverseProxy
	handle  apigateway.HandleFunc
}

func (h *Http) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("called [%s] /%s", r.Method, r.URL.Path[1:])
	requestRequest, err := FromHttp(r, w)
	if err != nil {
		badRequest(w)
		return
	}
	h.handle(requestRequest)
}

func FromHttp(r *http.Request, w http.ResponseWriter) (apigateway.Request, error) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	obj := apigateway.Request{
		Protocol:    "http",
		Context:     ctx,
		CtxCancel:   cancel,
		ClientIP:    r.RemoteAddr,
		HttpHeaders: r.Header,
		HttpMethod:  r.Method,
		URL:         "http://" + r.Host + r.RequestURI,
		Body:        r.Body,

		HttpResponseWriter: w,
	}
	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		obj.ClientIP = clientIP
	}
	return obj, nil
}

func badRequest(w http.ResponseWriter) {
	w.Write([]byte("bad request"))
}

func (h *Http) Start() error {
	logrus.Infof("start listening on %s", h.config.Addr)
	h.server = &http.Server{Addr: h.config.Addr, Handler: h}
	return h.server.ListenAndServe()
}

func (h *Http) Close() error {
	return h.server.Close()
}
