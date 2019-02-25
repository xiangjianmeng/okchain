// Copyright The go-okchain Authors 2018,  All rights reserved.

package log

import (
	"fmt"

	"github.com/op/go-logging"
)

const (
	pkgLogID     = "log"
	defaultLevel = logging.WARNING
)

// Level defines all available log levels for log messages.

const (
	CRITICAL logging.Level = logging.CRITICAL
	ERROR                  = logging.ERROR
	WARNING                = logging.WARNING
	NOTICE                 = logging.NOTICE
	INFO                   = logging.INFO
	DEBUG                  = logging.DEBUG
)

var ansiGray = "\033[0;37m"
var ansiBlue = "\033[0;34m"

// LogFormats defines formats for logging (i.e. "color")
var LogFormats = map[string]string{
	"nocolor": "%{time:2006-01-02 15:04:05.000000} %{level} %{module} %{shortfile}: %{message}",
	"color": ansiGray + "%{time:15:04:05.000} %{color}%{level:5.5s} " + ansiBlue +
		"%{module:10.10s}: %{color:reset}%{message} " + ansiGray + "%{shortfile}%{color:reset}",
	"verbose1": "%{time:01-02 15:04:05.000} %{level:.4s} [%{module}] [%{shortfile}] %{shortfunc}: %{message}",
}

//alias to go-logging's logging.Logger
type Logger = logging.Logger

func Debug(args ...interface{}) {
	logger.Debug(args...)
}
func Debugf(format string, args ...interface{}) {
	logger.Debugf(appendGid(format), args...)
}
func Error(args ...interface{}) {
	logger.Error(args...)
}
func Errorf(format string, args ...interface{}) {
	logger.Errorf(appendGid(format), args...)
}
func Info(args ...interface{}) {
	logger.Info(args...)
}
func Infof(format string, args ...interface{}) {
	logger.Infof(appendGid(format), args...)
}
func Warning(args ...interface{}) {
	logger.Warning(args...)
}
func Warningf(format string, args ...interface{}) {
	logger.Warningf(appendGid(format), args...)
}

func Critical(args ...interface{}) {
	logger.Critical(args...)
}
func Criticalf(format string, args ...interface{}) {
	logger.Criticalf(appendGid(format), args...)
}

func appendGid(format string) string {
	return fmt.Sprintf("%s [%s]->", format, gid)
}

type Lazy struct {
	Fn interface{}
}
