// Copyright The go-okchain Authors 2018,  All rights reserved.

package log

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
)

type goRoutineID int

var goroutineSpace = []byte("goroutine ")

var littleBuf = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 64)
		return &buf
	},
}

var gid goRoutineID = 0

func (base goRoutineID) String() string {
	bp := littleBuf.Get().(*[]byte)
	defer littleBuf.Put(bp)
	b := *bp
	b = b[:runtime.Stack(b, false)]
	// Parse the 4707 out of "goroutine 4707 ["
	b = bytes.TrimPrefix(b, goroutineSpace)
	i := bytes.IndexByte(b, ' ')
	if i < 0 {
		return "NOTAVAL"
	}
	b = b[:i]

	// always check if it was valid
	s := string(b)
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return "WRFMT"
	}

	// 0 mean "used as it like"
	if int(base) == 0 {
		return s
	} else {
		return strconv.FormatUint(n, int(base))
	}
}
