package apigateway

import (
	"context"
	"io"
	"net/http"
)

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
	Addr            string
	DestinationName string   `mapstructure:"destination"`
	MiddlewareNames []string `mapstructure:"middlewares"`

	Destination *Backend     `mapstructure:"-"`
	Middlewares []Middleware `mapstructure:"-"`
}

type Backend struct {
	Name          string
	Protocol      string
	Url           string
	DiscoveryType string `mapstructure:"discovery-type"`
}

type Middleware interface {
	Process(Request *Request) error
}

type Request struct {
	Protocol  string
	Context   context.Context
	CtxCancel context.CancelFunc
	ClientIP  string

	URL string

	Body               io.ReadCloser
	HttpHeaders        http.Header
	HttpMethod         string
	HttpResponseWriter http.ResponseWriter
}

type HandleFunc func(request Request)
