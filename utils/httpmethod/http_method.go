package httpmethod

import (
	"fmt"
	"net/http"
	"reflect"
)

const (
	MethodGet     uint8 = 1
	MethodPost    uint8 = 2
	MethodPut     uint8 = 4
	MethodDelete  uint8 = 8
	MethodPatch   uint8 = 16
	MethodOptions uint8 = 32
	MethodHead    uint8 = 64
	MethodTrace   uint8 = 128
	All                 = MethodGet | MethodPost | MethodPut | MethodDelete | MethodPatch | MethodOptions | MethodHead | MethodTrace
	Restful             = MethodGet | MethodPost | MethodPut | MethodDelete
)

var methodMap = map[uint8]string{
	MethodGet:     http.MethodGet,
	MethodPost:    http.MethodPost,
	MethodPut:     http.MethodPut,
	MethodDelete:  http.MethodDelete,
	MethodPatch:   http.MethodPatch,
	MethodOptions: http.MethodOptions,
	MethodHead:    http.MethodHead,
	MethodTrace:   http.MethodTrace,
}

func toMethod(method uint8) []string {
	var methods []string
	for k, v := range methodMap {
		if method&k != 0 {
			methods = append(methods, v)
		}
	}

	return methods
}

func Normalize(method any) ([]string, error) {
	t := reflect.TypeOf(method)
	switch t.Kind() {
	case reflect.String:
		return []string{method.(string)}, nil
	case reflect.Uint8:
		return toMethod(method.(uint8)), nil
	default:
		return nil, fmt.Errorf("invalid method type: %v", method)
	}
}
