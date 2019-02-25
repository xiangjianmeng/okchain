// Copyright The go-okchain Authors 2018,  All rights reserved.

package log

import (
	"testing"
)

func TestLog(t *testing.T) {
	var log = MustGetLogger("logtest")
	log.Error("test1")
	log.Errorf("test%d", 1)
	log.Debug("test", 2)
	log.Warningf("test%d", 3)
}
