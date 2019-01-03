package entrypoint

import (
	"testing"
	. "github.com/k3rn3l-p4n1c/apigateway"
	"github.com/stretchr/testify/assert"
	"bytes"
	"io/ioutil"
	"net/http"
	"time"
)

const host = "localhost:9090"

func TestHttpServer(t *testing.T) {
	config := &EntryPoint{
		Protocol: "http",
		Enabled:  &True,
		Addr:     host,
	}
	timeout := time.Duration(2 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	channel := make(chan *Request, 2)
	s, err := New(config, func(request *Request) *Response {
		body, _ := ioutil.ReadAll(request.Body)
		request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		channel <- request
		return &Response{
			Protocol:    "http",
			Body:        ioutil.NopCloser(bytes.NewBufferString("hi")),
			HttpStatus:  200,
			HttpHeaders: http.Header{},
		}
	})
	assert.NoError(t, err, "error in instantiating from http server")
	go s.Start()
	defer s.Close()

	t.Run("TestGetRequest", func(t *testing.T) {
		r, err := client.Get("http://" + host)
		if assert.NoError(t, err, "error while sending http request to server") {
			if assert.NotEqual(t, nil, r, "http response is nil") {
				request := <-channel
				assert.Equal(t, "http", request.Protocol)
				assert.Equal(t, "http://"+host+"/", request.URL)
				assert.Equal(t, "GET", request.HttpMethod)
			}
		}
	})

	t.Run("TestPostRequest", func(t *testing.T) {
		r, err := client.Post("http://"+host+"/url", "plain/txt", bytes.NewBufferString("body"))
		if assert.NoError(t, err, "error while sending http request to server") {
			if assert.NotEqual(t, nil, r, "http response is nil") {
				request := <-channel
				assert.Equal(t, "http", request.Protocol)
				assert.Equal(t, "http://"+host+"/url", request.URL)
				assert.Equal(t, "POST", request.HttpMethod)
				body, err := ioutil.ReadAll(request.Body)
				if assert.NoError(t, err, "fail to read request body") {
					assert.Equal(t, "body", string(body))
				}

			}
		}
	})
}
