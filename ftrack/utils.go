package ftrack

import (
	"golang.org/x/text/unicode/norm"
	"strings"
)

type uriParameter struct {
	key   string
	value string
}

func NormalizeString(str string) string {
	return norm.NFC.String(str)
}

func encodeUriParameters(data ...uriParameter) string {
	var parts []string
	for _, p := range data {
		parts = append(parts, strings.Join([]string{p.key, p.value}, "="))
	}
	return strings.Join(parts, "&")
}
