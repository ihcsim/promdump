package log

import (
	"io"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Logger encapsulates the underlying logging library.
type Logger struct {
	log.Logger
}

// New returns a new contextual logger. Log lines will be written to out.
func New(logLevel string, out io.Writer) *Logger {
	logger := log.NewLogfmtLogger(out)
	logger = log.With(logger,
		"time", log.TimestampFormat(now, time.RFC3339),
		"caller", log.DefaultCaller,
	)

	var opt level.Option
	switch strings.ToLower(logLevel) {
	case level.DebugValue().String():
		opt = level.AllowDebug()
	case level.ErrorValue().String():
		opt = level.AllowError()
	case level.WarnValue().String():
		opt = level.AllowWarn()
	case "all":
		opt = level.AllowAll()
	case "none":
		opt = level.AllowNone()
	case level.InfoValue().String():
		fallthrough
	default:
		opt = level.AllowInfo()
	}
	logger = level.NewFilter(logger, opt)

	return &Logger{logger}
}

func now() time.Time {
	return time.Now()
}

// With returns a new contextual logger with keyvals prepended to those passed
// to calls to Log. See https://pkg.go.dev/github.com/go-kit/kit/log#With
func (l *Logger) With(keyvals ...interface{}) *Logger {
	l.Logger = log.With(l.Logger, keyvals...)
	return l
}
