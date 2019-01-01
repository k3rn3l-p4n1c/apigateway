package engine

import (
	"testing"
	. "github.com/k3rn3l-p4n1c/apigateway"
	"github.com/stretchr/testify/assert"
)

func TestFrontendMatch(t *testing.T) {
	t.Run("TestProtocol", func(t *testing.T) {
		frontend := &Frontend {
			Protocol: "http",
		}
		request := &Request{
			Protocol: "grpc",
		}
		result := isMatch(frontend, request)
		assert.False(t, result, "grpc request should not match with http")
	})
	t.Run("TestHost", func(t *testing.T) {
		frontend := &Frontend {
			Protocol: "http",
			Host:     "app1.example.com",
		}
		request := &Request{
			Protocol: "http",
			URL: "http://app1.example.com/endpoint",
		}
		result := isMatch(frontend, request)
		assert.True(t, result, "host should matche")
	})
	t.Run("TestInvalidHost", func(t *testing.T) {
		frontend := &Frontend {
			Protocol: "http",
			Host:     "app1.example.com",
		}
		request := &Request{
			Protocol: "http",
			URL: "http:app1.example.com/endpoint",
		}
		result := isMatch(frontend, request)
		assert.False(t, result, "host should identify as invalid")
	})
}

func TestFindFrontend(t *testing.T) {
	c := &Config{
		Frontend: []*Frontend{
			{
				Protocol: "http",
				Host:     "app1.example.com",
			},
			{
				Protocol: "grpc",
				Host:     "app2.example.com",
			},
			{
				Protocol: "http",
				Host:     "app3.example.com",
			},
		},
	}
	e := Engine{
		config: c,
	}
	request := &Request{
		Protocol: "http",
		URL: "http://app1.example.com/endpoint",
	}

	front, err := e.findFrontend(request)
	assert.NoError(t, err, "error in finding frontend")
	assert.Equal(t, front.Host, "app1.example.com")
}

