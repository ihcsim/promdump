package runtime

import (
	"io"

	"k8s.io/client-go/tools/remotecommand"
)

// Runtime implements the streaming.Runtime interface. It provides the
// streams for the exec, attach and port-forward commands.
type Runtime interface {
	Exec(containerID string, cmd []string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error
	Attach(containerID string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error
	PortForward(podSandboxID string, port int32, stream io.ReadWriteCloser) error
}

type containerd struct{}

// NewContainerd returns a new instance of containerd runtime.
func NewContainerd() Runtime {
	return &containerd{}
}

func (c *containerd) Exec(containerID string, cmd []string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error {
	return nil
}

func (c *containerd) Attach(containerID string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error {
	return nil
}

func (c *containerd) PortForward(podSandboxID string, port int32, stream io.ReadWriteCloser) error {
	return nil
}
