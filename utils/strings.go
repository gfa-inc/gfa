package utils

import "strings"

func LoopTrimSuffix(target string, suffix string) string {
	v := target
	for {
		if strings.HasSuffix(v, suffix) {
			v = strings.TrimSuffix(v, suffix)
		} else {
			break
		}
	}

	return v
}
