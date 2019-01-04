package engine

import (
	"testing"
	. "github.com/k3rn3l-p4n1c/apigateway"
	"github.com/stretchr/testify/assert"
)

func TestFrontendMatch(t *testing.T) {
	t.Run("TestProtocol", func(t *testing.T) {
		frontend := &Frontend{
			Protocol: "http",
		}
		request := &Request{
			Protocol: "grpc",
		}
		result := isMatch(frontend, request)
		assert.False(t, result, "grpc request should not match with http")
	})
	t.Run("TestHost", func(t *testing.T) {
		frontend := &Frontend{
			Protocol: "http",
			Match: []MatchCondition{{
				Host: "app1.example.com",
			}},
		}
		request := &Request{
			Protocol: "http",
			URL:      "http://app1.example.com/endpoint",
		}
		result := isMatch(frontend, request)
		assert.True(t, result, "host should matche")
	})
	t.Run("TestInvalidHost", func(t *testing.T) {
		frontend := &Frontend{
			Protocol: "http",
			Match: []MatchCondition{{
				Host: "app1.example.com",
			}},
		}
		request := &Request{
			Protocol: "http",
			URL:      "http:app1.example.com/endpoint",
		}
		result := isMatch(frontend, request)
		assert.False(t, result, "host should identify as invalid")
	})
}

func TestFindFrontend(t *testing.T) {
	c := &Config{
		Frontend: []*Frontend{
			{
				Id:       "1",
				Protocol: "http",
				Match: []MatchCondition{{
					Host: "app1.example.com",
					Query: map[string]string{"version":"1"},
				}},
			},
			{
				Id:       "2",
				Protocol: "http",
				Match: []MatchCondition{{
					Host: "app1.example.com",
					Query: map[string]string{"version":"2"},
				}},
			},
			{
				Id:       "3",
				Protocol: "grpc",
				Match: []MatchCondition{{
					Host: "app1.example.com",
				}},
			},
			{
				Id:       "4",
				Protocol: "http",
				Match: []MatchCondition{{
					Host: "app4.example.com",
				}, {
					Host: "app4example.com",
				}},
			},
		},
	}
	e := Engine{
		config: c,
	}

	t.Run("TestMatchWithHost", func(t *testing.T) {
		request := &Request{
			Protocol: "http",
			URL:      "http://app1.example.com/endpoint?version=1",
		}

		front, err := e.findFrontend(request)
		if assert.NoError(t, err, "error in finding frontend") {
			assert.Equal(t, front.Id, "1")
		}

	})

	t.Run("TestMatchWithQuery", func(t *testing.T) {
		request := &Request{
			Protocol: "http",
			URL:      "http://app1.example.com/endpoint?version=2",
		}

		front, err := e.findFrontend(request)
		if assert.NoError(t, err, "error in finding frontend") {
			assert.Equal(t, front.Id, "2")
		}

	})

	t.Run("TestMatchOrCondition", func(t *testing.T) {
		request := &Request{
			Protocol: "http",
			URL:      "http://app4.example.com/endpoint?version=2",
		}

		front, err := e.findFrontend(request)
		if assert.NoError(t, err, "error in finding frontend") {
			assert.Equal(t, front.Id, "4")
		}

		request = &Request{
			Protocol: "http",
			URL:      "http://app4example.com/endpoint?version=2",
		}

		front, err = e.findFrontend(request)
		if assert.NoError(t, err, "error in finding frontend") {
			assert.Equal(t, front.Id, "4")
		}
	})
}
