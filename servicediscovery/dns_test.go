package servicediscovery

import (
	"testing"
	. "github.com/k3rn3l-p4n1c/apigateway"
	"github.com/stretchr/testify/assert"
	"github.com/sirupsen/logrus"
)

func TestDiscoverByDomain(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	config := Discovery{
		Type: "dns",
		Url: "amazon.com",
	}
	amazonIps := []string{"176.32.98.166", "176.32.103.205","205.251.242.103"}

	if d,err := New(config); assert.NoError(t, err, "fail to create new service discovery") {
		ip, err := d.Get("")
		if assert.NoError(t, err, "error in getting ip") {
			assert.Contains(t, amazonIps, ip, "resolved ip is not in amazon ips")
		}
	}


}