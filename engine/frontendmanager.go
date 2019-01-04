package engine

import (
	"errors"
	. "github.com/k3rn3l-p4n1c/apigateway"
	"github.com/sirupsen/logrus"
	"net/url"
)

func (e Engine) findFrontend(r *Request) (*Frontend, error) {
	for _, frontendConfig := range e.config.Frontend {
		if isMatch(frontendConfig, r) {
			return frontendConfig, nil
		}
	}
	return nil, errors.New("frontend does not match request")
}

func isMatch(frontend *Frontend, r *Request) bool {
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
		for _, condition := range frontend.Match {
			if condition.Host != "" {
				if condition.Host != rUrl.Host{
					continue
				}
			}
			if len(condition.Header) > 0 {
				match := true
				for k, v := range condition.Header {
					if r.HttpHeaders.Get(k) != v {
						match = false
						break
					}
				}
				if !match {
					continue
				}
			}
			if condition.Method != "" {
				if condition.Method != r.HttpMethod {
					continue
				}
			}
			if len(condition.Query) > 0 {
				match := true
				for k, v := range condition.Query {
					if rUrl.Query().Get(k) != v {
						match = false
						break
					}
				}
				if !match {
					continue
				}
			}

			// is matched with one condition at least
			return true
		}

		// matched with no condition
		return false
	default:
		logrus.WithField("protocol", r.Protocol).Debug("findFrontend invalid protocol error")
	}
	return false
}
