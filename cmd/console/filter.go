// Copyright The go-okchain Authors 2018,  All rights reserved.

package console

import "strings"

func excludeOptions(args []string) []string {
	ret := make([]string, 0, len(args))
	for i := range args {
		if !strings.HasSuffix(args[i], ".") {
			ret = append(ret, args[i])
		} else {
			ret = append(ret, args[i][:len(args[i])-1])
		}
	}
	return ret
}
