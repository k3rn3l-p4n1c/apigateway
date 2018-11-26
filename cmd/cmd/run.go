package cmd

import (
	"github.com/spf13/cobra"
	"github.com/k3rn3l-p4n1c/apigateway/configuration"
	"github.com/sirupsen/logrus"
	"github.com/k3rn3l-p4n1c/apigateway/server"
	"sync"
	"os"
	"os/signal"
	"syscall"
	"github.com/k3rn3l-p4n1c/apigateway/servicediscovery"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run server",
	Run:   Run,
}

func Run(cmd *cobra.Command, args []string) {
	config, err := configuration.Load()
	if err != nil {
		logrus.WithError(err).Fatal("unable to load configurations.")
	}

	serviceDiscovery, err := servicediscovery.NewServiceDiscovery(config)

	var forServers sync.WaitGroup
	var servers []server.Server
	for _, serverConfig := range config.Servers {
		forServers.Add(1)
		apiGatewayServer, err := server.Factory(serverConfig, serviceDiscovery)
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

func init() {
}
