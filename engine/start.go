package engine

import (
	. "github.com/k3rn3l-p4n1c/apigateway"
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
	"github.com/spf13/viper"
	"github.com/fsnotify/fsnotify"
	"errors"
	"time"
)

const DefaultTimeout = 10 * time.Second

type Engine struct {
	viper       *viper.Viper
	config      *Config
	entryPoints map[string]entrypoint.Server
	doneSignal  chan struct{}
}

func NewEngine(v *viper.Viper) (*Engine, error) {
	engine := &Engine{
		entryPoints: make(map[string]entrypoint.Server),
		doneSignal:  make(chan struct{}),
		viper: v,
	}

	logrus.Debug("load config:", engine.viper.AllSettings())
	var config = Config{}

	err := engine.viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	err = engine.loadConfig(&config)
	if err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *Engine) OnConfigChange(_ fsnotify.Event) {
	logrus.Info("reloading config")
	var config = Config{}
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

func (e *Engine) loadConfig(c *Config) error {
	if len(c.EntryPoints) == 0 {
		return errors.New("no entrypoint is set")
	}
	if len(c.Frontend) == 0 {
		return errors.New("no frontend is set")
	}
	if len(c.Backend) == 0 {
		return errors.New("no backend is set")
	}

	name2service := make(map[string]*Backend)
	for _, backend := range c.Backend {
		name2service[backend.Name] = backend
		if backend.Timeout == 0 {
			backend.Timeout = DefaultTimeout
		}
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

	for _, entryPointConfig := range c.EntryPoints {
		_, err := entrypoint.New(entryPointConfig, func(request *Request) *Response { return nil })
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
				newEntryPoint, err := entrypoint.New(entryPointConfig, e.Handle)
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
			if entryPointConfig.Enabled == nil {
				entryPointConfig.Enabled = &True
			}
			if !*entryPointConfig.Enabled {
				logrus.Warnf("entry point %s is not enabled", entryPointConfig.Protocol)
				continue
			}

			entryPoint, err := entrypoint.New(entryPointConfig, e.Handle)
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

func (e *Engine) Handle(request *Request) (resp *Response) {
	frontend, err := e.findFrontend(request)
	if err != nil {
		logrus.WithError(err).Info("error in finding frontend")
		return &Response{
			HttpStatus: http.StatusInternalServerError,
			Body:       ioutil.NopCloser(bytes.NewBufferString("error in finding frontend")),
		}
	}

	if frontend.Destination == nil {
		if err != nil {
			return &Response{
				HttpStatus: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewBufferString("backend is nil")),
			}
		}
	}

	if len(frontend.Middlewares) > 0 {
		resp, err = frontend.Middlewares[0].Handle(request)
		if err != nil {
			logrus.WithError(err).Error("error in middleware")
			return &Response{
				HttpStatus: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewBufferString("apigateway internal error")),
			}
		}
		if request.Context.Err() != nil {
			logrus.WithError(request.Context.Err()).Debug("error in context when processing request")
			return &Response{
				HttpStatus: http.StatusGatewayTimeout,
				Body:       ioutil.NopCloser(bytes.NewBufferString("timeout exceeded")),
			}
		}
	} else {
		resp, err = frontend.Destination.ReverseProxy.Handle(request)
		logrus.Debug("no middleware")
		if err != nil {
			logrus.WithError(err).Error("error in reverse proxy")
			return &Response{
				HttpStatus: http.StatusInternalServerError,
				Body:       ioutil.NopCloser(bytes.NewBufferString("apigateway internal error")),
			}
		}
	}
	return
}
