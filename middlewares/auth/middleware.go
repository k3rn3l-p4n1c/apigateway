package auth

import (
	. "github.com/k3rn3l-p4n1c/apigateway"
	"bytes"
	"net/http"
	"io/ioutil"
	"github.com/sirupsen/logrus"
)

type Auth struct {
	nextHandler Handler
}

func (a Auth) Handle(request *Request) (*Response, error) {
	authHeader, ok := request.HttpHeaders["authorization"]
	if !ok || len(authHeader) < 1 {
		writer := ioutil.NopCloser(bytes.NewBufferString("403 forbidden"))
		return &Response{
			HttpStatus: http.StatusForbidden,
			Body:       writer,
		}, nil
	}
	if request.HttpHeaders["authorization"][0] == "123123" {
		writer := ioutil.NopCloser(bytes.NewBufferString("403 forbidden"))
		return &Response{
			HttpStatus: http.StatusForbidden,
			Body:       writer,
		}, nil
	}
	logrus.Info("Auth middleware")
	resp, _ := a.nextHandler.Handle(request)
	return resp, nil
}

func (a Auth) SetNext(handler Handler) {
	a.nextHandler = handler
}