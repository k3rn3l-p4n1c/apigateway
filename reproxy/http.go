package reproxy

import (
	"net/http"
	"strings"
	"context"
	"github.com/sirupsen/logrus"
	"io"
	"github.com/k3rn3l-p4n1c/apigateway/servicediscovery"
	"time"
	"sync"
	"net/http/httputil"
	"github.com/k3rn3l-p4n1c/apigateway"
)

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; http://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

type HttpReverseProxy struct {
	serviceDiscovery *servicediscovery.ServiceDiscovery
	FlushInterval    time.Duration
	BufferPool       httputil.BufferPool
}

func NewHttpReverseProxy(serviceDiscovery *servicediscovery.ServiceDiscovery) *HttpReverseProxy {
	return &HttpReverseProxy{
		serviceDiscovery: serviceDiscovery,
	}
}

func (p HttpReverseProxy) Server(Request apigateway.Request, backend *apigateway.Backend) {
	logrus.Debug("proxying http")
	transport := http.DefaultTransport
	ctx := Request.Context
	if cn, ok := Request.HttpResponseWriter.(http.CloseNotifier); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		notifyChan := cn.CloseNotify()
		go func() {
			select {
			case <-notifyChan:
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	outReq, _ := http.NewRequest(Request.HttpMethod, Request.URL, Request.Body)
	outReq = outReq.WithContext(ctx)
	outReq.Header = cloneHeader(Request.HttpHeaders)

	err := p.director(backend, outReq)
	if err != nil {
		logrus.WithError(err).Error("unable to direct request")
	}
	outReq.Close = false

	removeConnectionHeaders(outReq.Header)

	// Remove hop-by-hop headers to the backend. Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.
	for _, h := range hopHeaders {
		if outReq.Header.Get(h) != "" {
			outReq.Header.Del(h)
		}
	}

	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := outReq.Header["X-Forwarded-For"]; ok {
		Request.ClientIP = strings.Join(prior, ", ") + ", " + Request.ClientIP
	}
	outReq.Header.Set("X-Forwarded-For", Request.ClientIP)

	res, err := transport.RoundTrip(outReq)
	if err != nil {
		logrus.Infof("http: proxy error: %v", err)
		Request.HttpResponseWriter.WriteHeader(http.StatusBadGateway)
		return
	}

	removeConnectionHeaders(res.Header)

	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	copyHeader(Request.HttpResponseWriter.Header(), res.Header)

	// The "Trailer" header isn't included in the Transport's response,
	// at least for *http.Transport. Build it up from Trailer.
	announcedTrailers := len(res.Trailer)
	if announcedTrailers > 0 {
		trailerKeys := make([]string, 0, len(res.Trailer))
		for k := range res.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		Request.HttpResponseWriter.Header().Add("Trailer", strings.Join(trailerKeys, ", "))
	}

	Request.HttpResponseWriter.WriteHeader(res.StatusCode)
	if len(res.Trailer) > 0 {
		// Force chunking if we saw a response trailer.
		// This prevents net/http from calculating the length for short
		// bodies and adding a Content-Length.
		if fl, ok := Request.HttpResponseWriter.(http.Flusher); ok {
			fl.Flush()
		}
	}
	p.copyResponse(Request.HttpResponseWriter, res.Body)
	res.Body.Close() // close now, instead of defer, to populate res.Trailer

	if len(res.Trailer) == announcedTrailers {
		copyHeader(Request.HttpResponseWriter.Header(), res.Trailer)
		return
	}

	for k, vv := range res.Trailer {
		k = http.TrailerPrefix + k
		for _, v := range vv {
			Request.HttpResponseWriter.Header().Add(k, v)
		}
	}
}

func cloneHeader(h http.Header) http.Header {
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}
	return h2
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// removeConnectionHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 2616, section 14.10.
func removeConnectionHeaders(h http.Header) {
	if c := h.Get("Connection"); c != "" {
		for _, f := range strings.Split(c, ",") {
			if f = strings.TrimSpace(f); f != "" {
				h.Del(f)
			}
		}
	}
}

func (p HttpReverseProxy) director(backend *apigateway.Backend, outReq *http.Request) error {
	scheme, host, path, err := p.serviceDiscovery.Get(backend)
	if err != nil {
		return err
	}
	outReq.URL.Scheme = scheme
	outReq.URL.Host = host
	outReq.URL.Path = singleJoiningSlash(path, outReq.URL.Path)
	//if request. == "" || outReq.URL.RawQuery == "" {
	//	outReq.URL.RawQuery = targetQuery + outReq.URL.RawQuery
	//} else {
	//	outReq.URL.RawQuery = targetQuery + "&" + outReq.URL.RawQuery
	//}
	if _, ok := outReq.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		outReq.Header.Set("User-Agent", "")
	}
	return nil
}

type writeFlusher interface {
	io.Writer
	http.Flusher
}

func (p HttpReverseProxy) copyResponse(dst io.Writer, src io.Reader) {
	if p.FlushInterval != 0 {
		if wf, ok := dst.(writeFlusher); ok {
			mlw := &maxLatencyWriter{
				dst:     wf,
				latency: p.FlushInterval,
				done:    make(chan bool),
			}
			go mlw.flushLoop()
			defer mlw.stop()
			dst = mlw
		}
	}

	var buf []byte
	if p.BufferPool != nil {
		buf = p.BufferPool.Get()
	}
	p.copyBuffer(dst, src, buf)
	if p.BufferPool != nil {
		p.BufferPool.Put(buf)
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
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

func (p *HttpReverseProxy) copyBuffer(dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	if len(buf) == 0 {
		buf = make([]byte, 32*1024)
	}
	var written int64
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
