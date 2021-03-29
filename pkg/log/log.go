package log

import (
	"io"

	"github.com/go-kit/kit/log"
)

// Logger encapsulates the underlying logging library.
type Logger struct {
	log.Logger
}

// New returns a new contextual logger. Log lines will be written to out.
func New(out io.Writer) *Logger {
	logger := log.NewLogfmtLogger(out)
	logger = log.With(logger,
		"timestamp", log.DefaultTimestamp,
		"caller", log.DefaultCaller)
	return &Logger{logger}
}

// With returns a new contextual logger with keyvals prepended to those passed
// to calls to Log. See https://pkg.go.dev/github.com/go-kit/kit/log#With
func (l *Logger) With(keyvals ...interface{}) *Logger {
	l.Logger = log.With(l.Logger, keyvals)
	return l
}
