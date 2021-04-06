package log

import (
	"io"
	"time"

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
		"time", log.TimestampFormat(now, time.RFC3339),
		"caller", log.DefaultCaller,
	)
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
