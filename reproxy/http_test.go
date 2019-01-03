package reproxy

import (
	"testing"
	"github.com/stretchr/testify/assert"
	. "github.com/k3rn3l-p4n1c/apigateway"
	"time"
	"context"
	"net/http"
	"bytes"
	"io/ioutil"
	"encoding/json"
)

func TestHttpReverseProxy(t *testing.T) {
	backend := &Backend{
		Name:     "reqres",
		Protocol: "http",
		Discovery: Discovery{
			Type: "static",
			Url:  "https://reqres.in/api",
		},
		Timeout: 5 * time.Second,
	}
	r, err := New(backend)
	if assert.NoError(t, err, "error in instantiating reverse proxy") {
		backend.ReverseProxy = r
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		response, err := r.Handle(&Request{
			Protocol:  "http",
			Context:   ctx,
			CtxCancel: cancel,
			ClientIP:  "192.168.0.1",

			URL: "http://localhost:9090/users?page=2",

			Body:        ioutil.NopCloser(bytes.NewBufferString("")),
			HttpHeaders: http.Header{},
			HttpMethod:  "GET",
		})
		if assert.NoError(t, err, "error in reverse proxy handle") {
			if assert.Equal(t, 200, response.HttpStatus, "response status is no ok") {
				respBody := make(map[string]interface{})
				respBodyJson, err := ioutil.ReadAll(response.Body)
				if assert.NoError(t, err, "error in reading body") {
					json.Unmarshal(respBodyJson, &respBody)
					assert.Equal(t, 2.0, respBody["page"], "reponse is wrong")
				}
			}
		}
	}
}
