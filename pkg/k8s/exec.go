package k8s

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecPod issues an exec request to execute the given command to a particular
// pod.
func (c *Clientset) ExecPod(command []string) error {
	var (
		ns             = c.config.GetString("namespace")
		pod            = c.config.GetString("pod")
		container      = c.config.GetString("container")
		requestTimeout = c.config.GetDuration("request-timeout")
		startTime      = c.config.GetTime("start-time")
		endTime        = c.config.GetTime("end-time")

		stdin  = true
		stdout = true
		stderr = true
		tty    = true
	)

	c.logger.Log("message", "sending exec request",
		"namespace", ns,
		"pod", pod,
		"container", container,
		"start-time", startTime,
		"end-time", endTime)

	execRequest := c.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(ns).
		Name(pod).
		SubResource("exec").
		Timeout(requestTimeout)

	execRequest.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   []string{"/bin/sh"},
		Stdin:     stdin,
		Stdout:    stdout,
		Stderr:    stderr,
		TTY:       tty,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.k8sConfig, "POST", execRequest.URL())
	if err != nil {
		return fmt.Errorf("failed to set up executor: %w", err)
	}

	if err := exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    tty,
	}); err != nil {
		return fmt.Errorf("failed to exec command: %w", err)
	}

	return nil
}
