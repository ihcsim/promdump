package runtime

import (
	"io"

	"k8s.io/client-go/tools/remotecommand"
)

// Runtime implements the streaming.Runtime interface. It provides the
// streams for the exec, attach and port-forward commands.
type Runtime struct {
}

// New returns a new instance of Runtime.
func New() *Runtime {
	return &Runtime{}
}

func (r *Runtime) Exec(containerID string, cmd []string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error {
	return nil
}

func (r *Runtime) Attach(containerID string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error {
	return nil
}

func (r *Runtime) PortForward(podSandboxID string, port int32, stream io.ReadWriteCloser) error {
	return nil
}
