// +build !windows

package util

import (
	"strings"
)

func CanonicalizePath(path string) string {

	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}
	return path

}

func CanonicalizeFilePath(filepath string) string {

	return filepath

}
