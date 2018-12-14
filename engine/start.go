package engine

import (
	"github.com/k3rn3l-p4n1c/apigateway"
	"github.com/k3rn3l-p4n1c/apigateway/entrypoint"
	"github.com/k3rn3l-p4n1c/apigateway/reproxy"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"github.com/k3rn3l-p4n1c/apigateway/middlewares"
	"net/http"
	"bytes"
	"io/ioutil"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/fsnotify/fsnotify"
)

type Engine struct {
	viper       *viper.Viper
	config      *apigateway.Config
	entryPoints map[string]entrypoint.Server
	doneSignal  chan struct{}
}

func NewEngine() (*Engine, error) {
	engine := &Engine{
		entryPoints: make(map[string]entrypoint.Server),
		doneSignal:  make(chan struct{}),
		viper: viper.New(),
	}

	engine.viper.SetConfigFile(configFilePath)

	engine.viper.OnConfigChange(engine.onConfigChange)
	engine.viper.WatchConfig()

	err := engine.viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("can't read v file error=(%v)", err)
	}
	engine.viper.SetDefault("log_level", "debug")
	setLogLevel(engine.viper.GetString("log_level"))
	logrus.Debug(engine.viper.AllSettings())
	var config = apigateway.Config{}

	err = engine.viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	err = engine.loadConfig(&config)
	if err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *Engine) onConfigChange(_ fsnotify.Event) {
	logrus.Info("reloading config")
	var config = apigateway.Config{}
	err := e.viper.Unmarshal(&config)
	if err != nil {
		logrus.WithError(err).Errorf("fail to change config")
		return
	}

	err = e.loadConfig(&config)
	if err != nil {
		logrus.WithError(err).Errorf("fail to reload config")
	}
}

func (e *Engine) Start() {
	interrupt := make(chan os.Signal, 1)
	finished := make(chan struct{}, 1)

	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		for range e.doneSignal {
		}
		finished <- struct{}{}
	}()

	select {
	case killSignal := <-interrupt:
		logrus.Info("got signal:", killSignal)

		for protocol, entryPoint := range e.entryPoints {
			logrus.Info("killing %s", protocol)
			entryPoint.Close()
		}
	case <-finished:
		logrus.Info("server stops working")
	}
}

func (e *Engine) loadConfig(c *apigateway.Config) error {
	name2service := make(map[string]*apigateway.Backend)
	for _, backend := range c.Backend {
		name2service[backend.Name] = backend
	}
	for _, frontend := range c.Frontend {
		frontend.Destination = name2service[frontend.DestinationName]
		if frontend.Destination == nil {
			return fmt.Errorf("no backend for name %s", frontend.DestinationName)
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
		var err error
		frontend.Destination.ReverseProxy, err = reproxy.New(frontend.Destination)
		if err != nil {
			return fmt.Errorf("fail to initialize reverse proxy for backend error=%v", err)
		}
		if len(frontend.Middlewares) > 0 {
			frontend.Middlewares[len(frontend.Middlewares)-1].SetNext(frontend.Destination.ReverseProxy)
		}
	}

	if len(c.EntryPoints) == 0 {
		return errors.New("no entrypoint is set")
	}

	for _, entryPointConfig := range c.EntryPoints {
		_, err := entrypoint.Factory(entryPointConfig, func(request *apigateway.Request) *apigateway.Response { return nil })
		if err != nil {
			logrus.WithError(err).Errorf("error in initializing server %s", entryPointConfig.Protocol)
			return fmt.Errorf("error in initializing server %s. error=%v", entryPointConfig.Protocol, err)
		}
	}

	e.config = c
	// ok
	for _, entryPointConfig := range c.EntryPoints {
		entryPoint, exists := e.entryPoints[entryPointConfig.Protocol]
		if exists {
			if entryPoint.EqualConfig(entryPointConfig) {
				logrus.Infof("no need to reload %s entry point", entryPointConfig.Protocol)
				continue
			} else {
				if entryPointConfig.Enabled != nil && *entryPointConfig.Enabled == false {
					logrus.Infof("killing &s", entryPointConfig.Protocol)
					entryPoint.Close()
					delete(e.entryPoints, entryPointConfig.Protocol)
				}
				newEntryPoint, err := entrypoint.Factory(entryPointConfig, e.Handle)
				if err != nil {
					logrus.WithError(err).Errorf("unable to create %s server.", entryPointConfig.Protocol)
					continue
				}
				entryPoint.Close()
				newEntryPoint.Start()
				e.entryPoints[entryPointConfig.Protocol] = newEntryPoint
				go func() {
					defer entryPoint.Close()
					err = entryPoint.Start()
					logrus.WithError(err).Info("server is shutting down.")
					e.doneSignal <- struct{}{}
				}()

			}
		} else {
			true := true
			if entryPointConfig.Enabled == nil {
				entryPointConfig.Enabled = &true
			}
			if !*entryPointConfig.Enabled {
				logrus.Warnf("entry point %s is not enabled", entryPointConfig.Protocol)
				continue
			}

			entryPoint, err := entrypoint.Factory(entryPointConfig, e.Handle)
			if err != nil {
				logrus.WithError(err).Errorf("unable to create %s server.", entryPointConfig.Protocol)
				continue
			}

			e.entryPoints[entryPointConfig.Protocol] = entryPoint

			go func() {
				defer entryPoint.Close()
				err = entryPoint.Start()
				logrus.WithError(err).Info("server is shutting down.")
				e.doneSignal <- struct{}{}
			}()
		}
	}
	return nil
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
