package k8s

import (
	"bytes"
	"io"
)

// ReadCloser implements the io.ReadCloser interface. It provides a Close()
// method to the io.Reader type, so that callers like k8s.ExecPod() can close
// the stdin stream when done.
type ReadCloser struct {
	*bytes.Buffer
}

func NewReadCloser(r io.Reader) (*ReadCloser, error) {
	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}

	return &ReadCloser{buf}, nil
}

// Close closes the underlying buffer of r by resetting it.
func (r *ReadCloser) Close() error {
	r.Reset()
	return nil
}
