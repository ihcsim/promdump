package fake

import (
	"net/url"

	"k8s.io/client-go/tools/remotecommand"
)

// Executor implements the remotecommand.Executor interface for testing
// purpose. See https://pkg.go.dev/k8s.io/client-go/tools/remotecommand#Executor.
type Executor struct {
	ServerURL *url.URL
}

// NewExecutor returns a new instance of Executor.
func NewExecutor(serverURL *url.URL) *Executor {
	return &Executor{serverURL}
}

// Stream provides the implementation to satisfy the remotecommand.Executor
// interface.
func (f *Executor) Stream(options remotecommand.StreamOptions) error {
	return nil
}
