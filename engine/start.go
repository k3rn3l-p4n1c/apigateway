package engine

import (
	"github.com/k3rn3l-p4n1c/apigateway"
	"github.com/k3rn3l-p4n1c/apigateway/entrypoint"
	"github.com/k3rn3l-p4n1c/apigateway/reproxy"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"github.com/k3rn3l-p4n1c/apigateway/middlewares"
	"net/http"
	"bytes"
	"io/ioutil"
)

type Engine struct {
	requestCh chan apigateway.Request
	config    *apigateway.Config
}

func (e *Engine) Start() {
	c, err := Load()
	if err != nil {
		logrus.WithError(err).Fatal("unable to load configurations.")
	} else {
		e.config = c
	}

	name2service := make(map[string]*apigateway.Backend)
	for _, backend := range c.Backend {
		name2service[backend.Name] = backend
	}
	for _, frontend := range c.Frontend {
		frontend.Destination = name2service[frontend.DestinationName]
		if frontend.Destination == nil {
			logrus.Fatalf("no backend for name %s", frontend.DestinationName)
		}
	}

	for _, frontend := range c.Frontend {
		for _, middlewareName := range frontend.MiddlewareNames {
			middleware := middlewares.Middlewares[middlewareName]
			if len(frontend.Middlewares) > 0 {
				frontend.Middlewares[len(frontend.Middlewares)-1].SetNext(middleware)
			}
			frontend.Middlewares = append(frontend.Middlewares, middleware)
		}
		frontend.Destination.ReverseProxy, err = reproxy.New(frontend.Destination)
		if err != nil {
			logrus.WithError(err).Fatal("fail to initialize reverse proxy for backend")
			return
		}
		if len(frontend.Middlewares) > 0 {
			frontend.Middlewares[len(frontend.Middlewares)-1].SetNext(frontend.Destination.ReverseProxy)
		}
	}

	e.requestCh = make(chan apigateway.Request, 100)

	var forServers sync.WaitGroup
	var servers []entrypoint.Server
	if len(e.config.EntryPoints) == 0 {
		logrus.Fatal("no entrypoint is set")
	}
	for _, entryPointConfig := range e.config.EntryPoints {
		true := true
		if entryPointConfig.Enabled == nil {
			entryPointConfig.Enabled = &true
		}
		if !*entryPointConfig.Enabled {
			logrus.Warnf("entry point %s is not enabled", entryPointConfig.Protocol)
			continue
		}

		forServers.Add(1)
		apiGatewayServer, err := entrypoint.Factory(entryPointConfig, e.Handle)
		servers = append(servers, apiGatewayServer)

		if err != nil {
			logrus.WithError(err).Fatal("unable to create server.")
		}

		go func() {
			defer apiGatewayServer.Close()
			err = apiGatewayServer.Start()
			logrus.WithError(err).Info("server is shutting down.")
			forServers.Done()
		}()
	}

	interrupt := make(chan os.Signal, 1)
	finished := make(chan struct{}, 1)

	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		forServers.Wait()
		finished <- struct{}{}
	}()

	select {
	case killSignal := <-interrupt:
		logrus.Info("got signal:", killSignal)

		for _, apiGatewayServer := range servers {
			apiGatewayServer.Close()
		}
	case <-finished:
		logrus.Info("server stops working")
	}
}

func (e *Engine) Handle(request *apigateway.Request) (resp *apigateway.Response) {
	frontend, err := e.findFrontend(request)
	if err != nil {
		logrus.WithError(err).Info("error in finding frontend")
		return &apigateway.Response{
			HttpStatus: http.StatusInternalServerError,
			Body:       ioutil.NopCloser(bytes.NewBufferString("error in finding frontend")),
		}
	}

	if frontend.Destination == nil {
		if err != nil {
			return &apigateway.Response{
				HttpStatus: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewBufferString("backend is nil")),
			}
		}
	}

	if len(frontend.Middlewares) > 0 {
		resp, err = frontend.Middlewares[0].Handle(request)
		if err != nil {
			logrus.WithError(err).Error("error in middleware")
			return &apigateway.Response{
				HttpStatus: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewBufferString("apigateway internal error")),
			}
		}
		if request.Context.Err() != nil {
			logrus.WithError(request.Context.Err()).Debug("error in context when processing request")
			return &apigateway.Response{
				HttpStatus: http.StatusGatewayTimeout,
				Body:       ioutil.NopCloser(bytes.NewBufferString("timeout exceeded")),
			}
		}
	} else {
		resp, err = frontend.Destination.ReverseProxy.Handle(request)
		logrus.Debug("no middleware")
		if err != nil {
			logrus.WithError(err).Error("error in reverse proxy")
			return &apigateway.Response{
				HttpStatus: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewBufferString("apigateway internal error")),
			}
		}
	}
	return
}
