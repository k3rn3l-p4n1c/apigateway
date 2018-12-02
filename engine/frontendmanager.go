package engine

import (
	"errors"
	"github.com/k3rn3l-p4n1c/apigateway"
	"net/url"
	"github.com/sirupsen/logrus"
)

func (e Engine) findFrontend(r apigateway.Request) (*apigateway.Frontend, error) {
	for _, frontendConfig := range e.config.Frontend {
		if isMatch(frontendConfig, r) {
			return frontendConfig, nil
		}
	}
	return nil, errors.New("frontend does not match request")
}

func isMatch(frontend *apigateway.Frontend, r apigateway.Request) bool {
	if r.Protocol != frontend.Protocol {
		return false
	}
	switch r.Protocol {
	case "http":
		rUrl, err := url.Parse(r.URL)
		if err != nil {
			logrus.WithError(err).Debug("findFrontend error in parsing url")
			return false
		}
		println(frontend.Addr, rUrl.Host, frontend.Addr == rUrl.Host)
		return frontend.Addr == rUrl.Host
	default:
		logrus.WithField("protocol", r.Protocol).Debug("findFrontend invalid protocol error")
	}
	return false
}

