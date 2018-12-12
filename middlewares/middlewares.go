package middlewares

import (
	"github.com/k3rn3l-p4n1c/apigateway"
	"github.com/k3rn3l-p4n1c/apigateway/middlewares/auth"
)

var Middlewares = map[string]apigateway.Middleware{
	"auth": auth.Auth{},
}
