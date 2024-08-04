package utils

import (
	"os"
)

type (
	EnvKey string
)

func (ek EnvKey) Read(or ...string) string {
	v := os.Getenv(string(ek))
	if v != "" {
		return v
	}
	if len(or) > 0 {
		return or[0]
	}
	return ""
}
