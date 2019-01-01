package engine

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/spf13/viper"
	"strings"
	"net/http"
	"io/ioutil"
	"time"
)

func TestInstantiating(t *testing.T) {
	v := viper.New()
	v.SetConfigFile("../config.yml")
	err := v.ReadInConfig()
	assert.NoError(t, err, "error in reading config")
	_, err = NewEngine(v)
	assert.NoError(t, err, "unable to instantiate Engine")
}

func TestSimpleHttpToHttpRequest(t *testing.T) {
	v := viper.New()
	config := `
frontend:
  - protocol: http
    enabled: true
    host: 127.0.0.1:9999
    destination: example

entryPoints:
  - protocol: http
    addr: 127.0.0.1:9999

backend:
  - name: example
    discovery:
      type: static
      url: http://example.com
    protocol: http
`
	v.SetConfigType("yml")
	err := v.ReadConfig(strings.NewReader(config))
	assert.NoError(t, err, "unable to read conf")

	v.Set("log_level", "debug")

	e, err := NewEngine(v)
	assert.NoError(t, err, "unable to instantiate Engine")

	go e.Start()
	time.Sleep(1 * time.Second)

	resp1, err := http.Get("http://127.0.0.1:9999")
	assert.NoError(t, err, "unable to get data from apigateway")
	assert.Equal(t, 200, resp1.StatusCode, "apigateway status code is not ok")

	resp2, err := http.Get("http://example.com")
	assert.NoError(t, err, "unable to get data from example.com")
	assert.Equal(t, 200, resp2.StatusCode, "example.com status code is not ok")

	body1, err := ioutil.ReadAll(resp1.Body)
	assert.NoError(t, err, "unable to read body from apigateway")

	body2, err := ioutil.ReadAll(resp2.Body)
	assert.NoError(t, err, "unable to read body from example.com")

	assert.Equal(t, string(body1), string(body2), "apigateway response body is not equal example.com")
}
