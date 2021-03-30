package fake

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/retry"
)

// Executor implements the remotecommand.Executor interface for testing
// purpose. See https://pkg.go.dev/k8s.io/client-go/tools/remotecommand#Executor.
type Executor struct {
	ServerURL     *url.URL
	SkipStreaming bool
}

// NewExecutor returns a new instance of Executor.
func NewExecutor(serverURL *url.URL, skipStreaming bool) *Executor {
	return &Executor{serverURL, skipStreaming}
}

// Stream provides the implementation to satisfy the remotecommand.Executor
// interface.
func (f *Executor) Stream(options remotecommand.StreamOptions) error {
	if f.SkipStreaming {
		return nil
	}

	var (
		resp *http.Response
		err  error
	)

	retriable := func(err error) bool {
		return err != nil
	}
	retry.OnError(retry.DefaultBackoff, retriable, func() error {
		url := fmt.Sprintf("%s", f.ServerURL)
		resp, err = http.DefaultClient.Post(url, "", nil)
		return err
	})
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)

	return nil
}
