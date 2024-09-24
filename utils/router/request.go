package router

import (
	"fmt"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/utils/http_method"
	"strings"
)

type RequestMatcher struct {
	Matcher
}

func NewRequestMatcher() *RequestMatcher {
	return &RequestMatcher{
		Matcher: *New(),
	}
}

func (rm *RequestMatcher) AddRoute(route string, method any) {
	basePath := config.GetString("server.base_path")
	if basePath != "" && !strings.HasPrefix(route, basePath) {
		route = strings.Join([]string{basePath, route}, "")
	}

	normalizedMethods, err := http_method.Normalize(method)
	if err != nil {
		logger.Error(err)
		return
	}

	for _, m := range normalizedMethods {
		rm.Matcher.AddRoute(genMatchKey(route, m))
	}
}

func (rm *RequestMatcher) AddRoutes(routes [][]any) {
	for _, v := range routes {
		if len(v) != 2 {
			logger.Panicf("Invalid match route: %v", v)
		}
		rm.AddRoute(v[0].(string), v[1])
	}
}

func (rm *RequestMatcher) Match(uri, method string) bool {
	matchKey := genMatchKey(uri, method)
	return rm.Matcher.Match(matchKey)
}

func genMatchKey(uri, method string) string {
	return fmt.Sprintf("%s#%s", uri, method)
}
