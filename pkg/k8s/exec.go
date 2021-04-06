package k8s

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

var deniedCreateExecErr = fmt.Errorf("no permissions to create exec subresource")

// ExecPod issues an exec request to execute the given command to a particular
// pod.
func (c *Clientset) ExecPod(command []string, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	readCloser, err := NewReadCloser(stdin)
	if err != nil {
		return err
	}
	defer readCloser.Close()

	var (
		ns             = c.config.GetString("namespace")
		pod            = c.config.GetString("pod")
		container      = c.config.GetString("container")
		requestTimeout = c.config.GetDuration("request-timeout")
		startTime      = c.config.GetTime("start-time")
		endTime        = c.config.GetTime("end-time")
	)

	_ = c.logger.Log("message", "sending exec request",
		"command", strings.Join(command, " "),
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

	execRequest = execRequest.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdin:     stdin != nil,
		Stdout:    stdout != nil,
		Stderr:    stderr != nil,
		TTY:       tty,
	}, scheme.ParameterCodec)

	exec, err := newExecutor(c.k8sConfig, "POST", execRequest.URL())
	if err != nil {
		return fmt.Errorf("failed to set up executor: %w", err)
	}

	if err := exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	}); err != nil {
		return fmt.Errorf("failed to exec command: %w", err)
	}

	return nil
}

// CanExec determines if the current user can create a exec subresource in the
// given pod.
func (c *Clientset) CanExec() error {
	var (
		ns      = c.config.GetString("namespace")
		timeout = c.config.GetDuration("request-timeout")
	)
	selfAccessReview := &authzv1.SelfSubjectAccessReview{
		Spec: authzv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authzv1.ResourceAttributes{
				Namespace:   ns,
				Verb:        "create",
				Group:       "",
				Resource:    "pods",
				Subresource: "exec",
				Name:        "",
			},
		},
	}

	_ = c.logger.Log("message", "checking for exec permissions",
		"namespace", ns,
		"request-timeout", timeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	response, err := c.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, selfAccessReview, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if !response.Status.Allowed {
		if response.Status.Reason != "" {
			return fmt.Errorf("%w. reason: %s", deniedCreateExecErr, response.Status.Reason)
		}
		return deniedCreateExecErr
	}

	_ = c.logger.Log("message", "confirmed exec permissions",
		"namespace", ns,
		"request-timeout", timeout)
	return nil
}

var newExecutor = func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
	return remotecommand.NewSPDYExecutor(config, method, url)
}
