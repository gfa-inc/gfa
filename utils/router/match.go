package router

import (
	"strings"
)

type Matcher struct {
	expectedRoutes map[string]struct{}
}

func New() *Matcher {
	return &Matcher{
		expectedRoutes: make(map[string]struct{}),
	}
}

func (m *Matcher) AddRoutes(routes []string) {
	for _, v := range routes {
		m.AddRoute(v)
	}
}

func (m *Matcher) AddRoute(route string) {
	m.expectedRoutes[route] = struct{}{}
}

func (m *Matcher) Match(uri string) bool {
	if uri == "" {
		return false
	}

	if _, ok := m.expectedRoutes[uri]; ok {
		return true
	}

	newUri := uri
	for {
		var key string
		index := strings.LastIndex(newUri, "/")
		if index >= 0 {
			newUri = newUri[0:index]
			key = newUri + "/*"
		} else {
			key = "*"
		}

		if _, ok := m.expectedRoutes[key]; ok {
			return true
		}

		if index < 0 {
			break
		}
	}

	return false
}
