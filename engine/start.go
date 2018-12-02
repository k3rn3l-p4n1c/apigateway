package engine

import (
	"github.com/sirupsen/logrus"
	"sync"
	"github.com/k3rn3l-p4n1c/apigateway/entrypoint"
	"os"
	"os/signal"
	"syscall"
	"github.com/k3rn3l-p4n1c/apigateway/reproxy"
	"github.com/k3rn3l-p4n1c/apigateway/servicediscovery"
	"github.com/k3rn3l-p4n1c/apigateway"
)

type Engine struct {
	requestCh    chan apigateway.Request
	config       *apigateway.Config
	reverseProxy reproxy.Interface
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

	sd := servicediscovery.NewServiceDiscovery(c)
	e.reverseProxy, _ = reproxy.NewReverseProxy(sd)

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

func (e *Engine) Handle(r apigateway.Request) {
	//if r.Context.Err() != nil {
	//	logrus.WithError(r.Context.Err()).Debug("error in context when processing request")
	//	continue
	//}

	frontend, err := e.findFrontend(r)
	if err != nil {
		logrus.WithError(err).Info("error in finding frontend")
		r.HttpResponseWriter.Write([]byte("error in finding frontend"))
		return
	}
	for _, middleware := range frontend.Middlewares {
		middleware.Process(&r)
	}
	if frontend.Destination == nil {
		r.HttpResponseWriter.Write([]byte("backend is nil"))
		return
	}
	e.reverseProxy.Serve(r, frontend.Destination)

}
