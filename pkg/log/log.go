package log

import (
	"io"

	"github.com/go-kit/kit/log"
)

// New returns a new contextual logger. Log lines will be written to out.
func New(out io.Writer) log.Logger {
	logger := log.NewLogfmtLogger(out)
	logger = log.With(logger,
		"timestamp", log.DefaultTimestamp,
		"caller", log.DefaultCaller)
	return logger
}
