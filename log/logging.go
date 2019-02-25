// Copyright The go-okchain Authors 2018,  All rights reserved.

package log

import (
	"io"
	"os"
	"regexp"
	"strings"

	"sync"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

// A logger to log logging logs!
var loggingLogger = logging.MustGetLogger("logging")

// The default logging level, in force until LoggingInit() is called or in
// case of configuration errors.
var loggingDefaultLevel = logging.INFO

//defaultFormat
var defaultFormat = "verbose1"

var (
	logger           *logging.Logger
	defaultOutput    *os.File
	modules          map[string]string // Holds the map of all modules and their respective log level
	peerStartModules map[string]string
	lock             sync.RWMutex
	once             sync.Once
)

// LoggingInit is a 'hook' called at the beginning of command processing to
// parse logging-related options specified either on the command-line or in
// config files.  Command-line options take precedence over config file
// options, and can also be passed as suitably-named environment variables. To
// change module logging levels at runtime call `logging.SetLevel(level,
// module)`.  To debug this routine include logging=debug as the first
// term of the logging specification.
func LoggingInit(command string) {
	// Parse the logging specification in the form
	//     [<module>[,<module>...]=]<level>[:[<module>[,<module>...]=]<level>...]
	defaultLevel := loggingDefaultLevel
	var err error
	spec := viper.GetString("logging_level")
	if spec == "" {
		spec = viper.GetString("logging." + command)
	}
	if spec != "" {
		fields := strings.Split(spec, ":")
		for _, field := range fields {
			split := strings.Split(field, "=")
			switch len(split) {
			case 1:
				// Default level
				defaultLevel, err = logging.LogLevel(field)
				if err != nil {
					loggingLogger.Warningf("Logging level '%s' not recognized, defaulting to %s : %s", field, loggingDefaultLevel, err)
					defaultLevel = loggingDefaultLevel // NB - 'defaultLevel' was overwritten
				}
			case 2:
				// <module>[,<module>...]=<level>
				if level, err := logging.LogLevel(split[1]); err != nil {
					loggingLogger.Warningf("Invalid logging level in '%s' ignored", field)
				} else if split[0] == "" {
					loggingLogger.Warningf("Invalid logging override specification '%s' ignored - no module specified", field)
				} else {
					modules := strings.Split(split[0], ",")
					for _, module := range modules {
						logging.SetLevel(level, module)
						loggingLogger.Debugf("Setting logging level for module '%s' to %s", module, level)
					}
				}
			default:
				loggingLogger.Warningf("Invalid logging override '%s' ignored; Missing ':' ?", field)
			}
		}
	}
	// Set the default logging level for all modules
	logging.SetLevel(defaultLevel, "")
	loggingLogger.Debugf("Setting default logging level to %s for command '%s'", defaultLevel, command)

}

// DefaultLoggingLevel returns the fallback value for loggers to use if parsing fails
func DefaultLoggingLevel() logging.Level {
	return loggingDefaultLevel
}

// Initiate 'leveled' logging to stderr.
func init() {

	logger = logging.MustGetLogger(pkgLogID)
	Reset()

	//format := logging.MustStringFormatter(
	//	"[stdout][%{time:15:04:05.000}][%{level:.4s}][%{pid}][%{module}][%{shortfunc}]: %{message}")
	//format := logging.MustStringFormatter(
	//	"%{time:01-02 15:04:05.000} %{level:.4s} [%{pid}] [%{module}] [%{shortfile}] %{shortfunc}: %{message}")
	//
	//DefaultBackend = &ModuleLeveled{
	//	//levels: make(map[string]logging.Level),
	//	Backend: logging.NewBackendFormatter(
	//		logging.NewLogBackend(os.Stderr, "", 0), format),
	//}
	//DefaultBackend.SetLevel(loggingDefaultLevel, "")
	//
	//logging.SetBackend(DefaultBackend)
}

// Reset sets to logging to the defaults defined in this package.
func Reset() {
	modules = make(map[string]string)
	lock = sync.RWMutex{}
	logFilePath := os.Getenv("OKCHAIN_PEER_LOGPATH")
	output, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		output = os.Stderr
	}

	InitBackend(SetFormat(LogFormats[defaultFormat]), output)
	InitFromSpec("")
}

// SetFormat sets the logging format.
func SetFormat(formatSpec string) logging.Formatter {
	if formatSpec == "" {
		formatSpec = LogFormats[defaultFormat]
	}
	return logging.MustStringFormatter(formatSpec)
}

// InitBackend sets up the logging backend based on
// the provided logging formatter and I/O writer.
func InitBackend(formatter logging.Formatter, output io.Writer) {
	backend := logging.NewLogBackend(output, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, formatter)
	logging.SetBackend(backendFormatter).SetLevel(defaultLevel, "")
}

// MustGetLogger is used in place of `logging.MustGetLogger` to allow us to
// store a map of all modules and submodules that have loggers in the system.
func MustGetLogger(module string) *Logger {
	l := logging.MustGetLogger(module)
	lock.Lock()
	defer lock.Unlock()
	modules[module] = GetModuleLevel(module)
	return (*Logger)(l)
}

// GetModuleLevel gets the current logging level for the specified module.
func GetModuleLevel(module string) string {
	// logging.GetLevel() returns the logging level for the module, if defined.
	// Otherwise, it returns the default logging level, as set by
	// `flogging/logging.go`.
	level := logging.GetLevel(module).String()
	return level
}

// SetModuleLevel sets the logging level for the modules that match the supplied
// regular expression. Can be used to dynamically change the log level for the
// module.
func SetModuleLevel(moduleRegExp string, level string) (string, error) {
	return setModuleLevel(moduleRegExp, level, true, false)
}

func setModuleLevel(moduleRegExp string, level string, isRegExp bool, revert bool) (string, error) {
	var re *regexp.Regexp
	logLevel, err := logging.LogLevel(level)
	if err != nil {
		logger.Warningf("Invalid logging level '%s' - ignored", level)
	} else {
		if !isRegExp || revert {
			logging.SetLevel(logLevel, moduleRegExp)
			logger.Debugf("Module '%s' logger enabled for log level '%s'", moduleRegExp, level)
		} else {
			re, err = regexp.Compile(moduleRegExp)
			if err != nil {
				logger.Warningf("Invalid regular expression: %s", moduleRegExp)
				return "", err
			}
			lock.Lock()
			defer lock.Unlock()
			for module := range modules {
				if re.MatchString(module) {
					logging.SetLevel(logging.Level(logLevel), module)
					modules[module] = logLevel.String()
					logger.Debugf("Module '%s' logger enabled for log level '%s'", module, logLevel)
				}
			}
		}
	}
	return logLevel.String(), err
}

// InitFromSpec initializes the logging based on the supplied spec. It is
// exposed externally so that consumers of the flogging package may parse their
// own logging specification. The logging specification has the following form:
//		[<module>[,<module>...]=]<level>[:[<module>[,<module>...]=]<level>...]
func InitFromSpec(spec string) string {
	levelAll := defaultLevel
	var err error

	if spec != "" {
		fields := strings.Split(spec, ":")
		for _, field := range fields {
			split := strings.Split(field, "=")
			switch len(split) {
			case 1:
				if levelAll, err = logging.LogLevel(field); err != nil {
					logger.Warningf("Logging level '%s' not recognized, defaulting to '%s': %s", field, defaultLevel, err)
					levelAll = defaultLevel // need to reset cause original value was overwritten
				}
			case 2:
				// <module>[,<module>...]=<level>
				levelSingle, err := logging.LogLevel(split[1])
				if err != nil {
					logger.Warningf("Invalid logging level in '%s' ignored", field)
					continue
				}

				if split[0] == "" {
					logger.Warningf("Invalid logging override specification '%s' ignored - no module specified", field)
				} else {
					modules := strings.Split(split[0], ",")
					for _, module := range modules {
						logger.Debugf("Setting logging level for module '%s' to '%s'", module, levelSingle)
						logging.SetLevel(levelSingle, module)
					}
				}
			default:
				logger.Warningf("Invalid logging override '%s' ignored - missing ':'?", field)
			}
		}
	}

	logging.SetLevel(levelAll, "") // set the logging level for all modules

	// iterate through modules to reload their level in the modules map based on
	// the new default level
	for k := range modules {
		MustGetLogger(k)
	}
	// register flogging logger in the modules map
	MustGetLogger(pkgLogID)

	return levelAll.String()
}
