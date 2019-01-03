package apigateway

import (
	"context"
	"io"
	"net/http"
	"time"
)

var True = true

type Config struct {
	EntryPoints []*EntryPoint
	Frontend    []*Frontend
	Backend     []*Backend
}

type EntryPoint struct {
	Protocol string
	Enabled  *bool
	Addr     string
}

type Frontend struct {
	Protocol        string
	Host            string
	DestinationName string   `mapstructure:"destination"`
	MiddlewareNames []string `mapstructure:"middlewares"`

	Destination *Backend     `mapstructure:"-"`
	Middlewares []Middleware `mapstructure:"-"`
}

type Backend struct {
	Name      string
	Protocol  string
	Discovery Discovery
	Timeout   time.Duration

	ReverseProxy ReverseProxy `mapstructure:"-"`
}

type Discovery struct {
	Type string
	Url  string
}

type Handler interface {
	Handle(request *Request) (*Response, error)
}

type ServiceDiscovery interface {
	Get(request *Request) (scheme string, host string, path string, err error)
}

type Middleware interface {
	Handler
	SetNext(handler Handler)
}

type ReverseProxy interface {
	Handler
}

type Request struct {
	Protocol  string
	Context   context.Context
	CtxCancel context.CancelFunc
	ClientIP  string

	URL string

	Body        io.ReadCloser
	HttpHeaders http.Header
	HttpMethod  string
}

type Response struct {
	Protocol    string
	Body        io.ReadCloser
	HttpStatus  int
	HttpHeaders http.Header
}

type HandleFunc func(request *Request) *Response
