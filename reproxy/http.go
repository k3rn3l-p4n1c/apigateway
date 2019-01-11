package reproxy

import (
	"context"
	. "github.com/k3rn3l-p4n1c/apigateway"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"bytes"
	"io/ioutil"
	"time"
	"net/http/httputil"
	"io"
	"sync"
	"net/url"
	"net"
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

var defaultDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
	DualStack: true,
}

var getDialFunc = func(serviceDiscovery ServiceDiscovery) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		colon := strings.IndexByte(addr, ':')
		port := addr[colon+1:]
		ip, err := serviceDiscovery.Get(addr)
		if err != nil {
			return nil, err
		}

		return defaultDialer.DialContext(ctx, network, ip+":"+port)
	}
}

type HttpReverseProxy struct {
	serviceDiscovery ServiceDiscovery
	backend          *Backend
	FlushInterval    time.Duration
	BufferPool       httputil.BufferPool
	transport        http.RoundTripper
}

func NewHttpReverseProxy(serviceDiscovery ServiceDiscovery, backend *Backend) (*HttpReverseProxy, error) {

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext:           getDialFunc(serviceDiscovery),
	}
	return &HttpReverseProxy{
		serviceDiscovery: serviceDiscovery,
		backend:          backend,
		transport:        transport,
	}, nil
}

func (p HttpReverseProxy) Handle(request *Request) (*Response, error) {
	logrus.Debug("proxying http")

	ctx := request.Context
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, p.backend.Timeout)
	defer cancel()
	go func() {
		select {
		case <-request.Context.Done():
			cancel()
		case <-ctx.Done():
		}
	}()

	outReq, _ := http.NewRequest(request.HttpMethod, request.URL, request.Body)
	outReq = outReq.WithContext(ctx)
	outReq.Header = cloneHeader(request.HttpHeaders)
	err := p.director(request, outReq)
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
		request.ClientIP = strings.Join(prior, ", ") + ", " + request.ClientIP
	}
	outReq.Header.Set("X-Forwarded-For", request.ClientIP)

	res, err := p.transport.RoundTrip(outReq)
	if err != nil {
		logrus.Infof("http: reproxy error: %v", err)
		//request.HttpResponseWriter.WriteHeader(http.StatusBadGateway)
		return &Response{
			Protocol:   "http",
			Body:       ioutil.NopCloser(bytes.NewBufferString("bad gateway")),
			HttpStatus: http.StatusBadGateway,
		}, err
	}

	finalResp := &Response{
		Protocol:    "http",
		HttpHeaders: make(http.Header),
		HttpStatus:  http.StatusBadGateway,
	}

	removeConnectionHeaders(res.Header)

	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	//copyHeader(request.HttpResponseWriter.Header(), res.Header)
	copyHeader(finalResp.HttpHeaders, res.Header)

	// The "Trailer" header isn't included in the Transport's response,
	// at least for *http.Transport. Build it up from Trailer.
	announcedTrailers := len(res.Trailer)
	if announcedTrailers > 0 {
		trailerKeys := make([]string, 0, len(res.Trailer))
		for k := range res.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		finalResp.HttpHeaders.Add("Trailer", strings.Join(trailerKeys, ", "))
		//request.HttpResponseWriter.Header().Add("Trailer", strings.Join(trailerKeys, ", "))
	}

	//request.HttpResponseWriter.WriteHeader(res.StatusCode)
	finalResp.HttpStatus = res.StatusCode
	if len(res.Trailer) > 0 {
		// Force chunking if we saw a response trailer.
		// This prevents net/http from calculating the length for short
		// bodies and adding a Content-Length.
		panic("WTF?")
		//if fl, ok := request.HttpResponseWriter.(http.Flusher); ok {
		//	fl.Flush()
		//}
	}

	//p.copyResponse(request.HttpResponseWriter, res.Body)

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logrus.WithError(err).Debug("error in reading request body")
	}

	finalResp.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	res.Body.Close() // close now, instead of defer, to populate res.Trailer

	if len(res.Trailer) == announcedTrailers {
		//copyHeader(request.HttpResponseWriter.Header(), res.Trailer)
		copyHeader(finalResp.HttpHeaders, res.Trailer)
		return finalResp, nil

	}

	for k, vv := range res.Trailer {
		k = http.TrailerPrefix + k
		for _, v := range vv {
			//request.HttpResponseWriter.Header().Add(k, v)
			finalResp.HttpHeaders.Add(k, v)
		}
	}
	return finalResp, nil
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

func (p HttpReverseProxy) director(incomingReq *Request, outReq *http.Request) error {
	incomingUrl, err := url.Parse(incomingReq.URL)
	if err != nil {
		return err
	}
	outReq.URL.Scheme = p.backend.Scheme

	if p.backend.ForwardHost {
		outReq.URL.Host = incomingUrl.Host
		outReq.Host = incomingUrl.Host
	} else {
		outReq.URL.Host = p.backend.Host
		outReq.Host = p.backend.Host
	}
	outReq.URL.Path = singleJoiningSlash(p.backend.Path, outReq.URL.Path)
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
