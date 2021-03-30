package k8s

import (
	"context"
	"fmt"
	"os"

	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	response, err := c.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, selfAccessReview, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if !response.Status.Allowed {
		msg := fmt.Sprintf("no permission to create pods/exec subresource in namespace:%s. ", ns)
		if response.Status.Reason != "" {
			msg += response.Status.Reason
		}

		if response.Status.EvaluationError != "" {
			msg += response.Status.EvaluationError
		}
		return fmt.Errorf(msg)
	}

	return nil
}
