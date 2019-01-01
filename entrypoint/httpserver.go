package entrypoint

import (
	"context"
	. "github.com/k3rn3l-p4n1c/apigateway"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
	"io"
	"sync"
)

type Http struct {
	config  *EntryPoint
	server  *http.Server
	proxies map[string]*httputil.ReverseProxy
	handle  HandleFunc

	FlushInterval time.Duration
	BufferPool    httputil.BufferPool
}

func (h *Http) Start() error {
	logrus.Infof("start listening on %s", h.config.Addr)
	h.server = &http.Server{Addr: h.config.Addr, Handler: h}
	return h.server.ListenAndServe()
}

func (h *Http) Close() error {
	return h.server.Close()
}

func (h *Http) EqualConfig(c *EntryPoint) bool {
	return c.Protocol == h.config.Protocol &&
		c.Enabled == h.config.Enabled &&
		c.Addr == h.config.Addr
}

func (h *Http) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("called [%s] /%s", r.Method, r.URL.Path[1:])
	requestRequest, err := FromHttp(r)
	if err != nil {
		badRequest(w)
		return
	}
	response := h.handle(requestRequest)
	logrus.Debugf("start writing response")
	h.WriteToHttp(w, response)
}

func FromHttp(r *http.Request) (*Request, error) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	obj := Request{
		Protocol:    "http",
		Context:     ctx,
		CtxCancel:   cancel,
		ClientIP:    r.RemoteAddr,
		HttpHeaders: r.Header,
		HttpMethod:  r.Method,
		URL:         "http://" + r.Host + r.RequestURI,
		Body:        r.Body,
	}
	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		obj.ClientIP = clientIP
	}
	return &obj, nil
}

func (h *Http) WriteToHttp(w http.ResponseWriter, response *Response) {
	copyHeader(w.Header(), response.HttpHeaders)
	h.copyResponse(w, response.Body)
	w.WriteHeader(response.HttpStatus)
}

func (h *Http) copyResponse(dst io.Writer, src io.ReadCloser) {
	if h.FlushInterval != 0 {
		if wf, ok := dst.(writeFlusher); ok {
			mlw := &maxLatencyWriter{
				dst:     wf,
				latency: h.FlushInterval,
				done:    make(chan bool),
			}
			go mlw.flushLoop()
			defer mlw.stop()
			dst = mlw
		}
	}

	var buf []byte
	if h.BufferPool != nil {
		buf = h.BufferPool.Get()
	}
	h.copyBuffer(dst, src, buf)
	if h.BufferPool != nil {
		h.BufferPool.Put(buf)
	}
}

func (h *Http) copyBuffer(dst io.Writer, src io.ReadCloser, buf []byte) (int64, error) {
	if len(buf) == 0 {
		buf = make([]byte, 32*1024)
	}
	var written int64
	//defer src.Close()
	for {
		nr, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF && rerr != context.Canceled {
			logrus.Infof("httputil: ReverseProxy read error during body copy: %v", rerr)
		}
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rerr != nil {
			return written, rerr
		}
	}
}

func badRequest(w http.ResponseWriter) {
	w.Write([]byte("bad request"))
}

type maxLatencyWriter struct {
	dst     writeFlusher
	latency time.Duration

	mu   sync.Mutex // protects Write + Flush
	done chan bool
}

func (m *maxLatencyWriter) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dst.Write(p)
}

var onExitFlushLoop func()

func (m *maxLatencyWriter) flushLoop() {
	t := time.NewTicker(m.latency)
	defer t.Stop()
	for {
		select {
		case <-m.done:
			if onExitFlushLoop != nil {
				onExitFlushLoop()
			}
			return
		case <-t.C:
			m.mu.Lock()
			m.dst.Flush()
			m.mu.Unlock()
		}
	}
}

func (m *maxLatencyWriter) stop() { m.done <- true }

type writeFlusher interface {
	io.Writer
	http.Flusher
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
