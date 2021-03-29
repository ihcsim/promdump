package log

import (
	"io"
	"sync"

	"github.com/go-kit/kit/log"
)

// Logger encapsulates the underlying logging library.
type Logger struct {
	log.Logger
	errs []error
	mux  sync.Mutex
}

// New returns a new contextual logger. Log lines will be written to out.
func New(out io.Writer) *Logger {
	logger := log.NewLogfmtLogger(out)
	logger = log.With(logger,
		"timestamp", log.DefaultTimestamp,
		"caller", log.DefaultCaller)
	return &Logger{logger, []error{}, sync.Mutex{}}
}

// With returns a new contextual logger with keyvals prepended to those passed
// to calls to Log. See https://pkg.go.dev/github.com/go-kit/kit/log#With
func (l *Logger) With(keyvals ...interface{}) *Logger {
	l.Logger = log.With(l.Logger, keyvals...)
	return l
}

// Log wraps the underlying Log() function and collects any errors returned
// by the wrapped function. Caller of logger decides how to handle the collectedi
// errors.
func (l *Logger) Log(keyvals ...interface{}) {
	if err := l.Logger.Log(keyvals...); err != nil {
		l.mux.Lock()
		defer l.mux.Unlock()
		l.errs = append(l.errs, err)
	}
}
