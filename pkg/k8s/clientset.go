package k8s

import (
	"fmt"
	"os"

	"github.com/ihcsim/promdump/pkg/config"
	"github.com/ihcsim/promdump/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// Clientset knows how to interact with a K8s cluster. It has a reference to
// user configuration stored in viper.
type Clientset struct {
	config    *config.Config
	k8sConfig *rest.Config
	logger    *log.Logger
	*kubernetes.Clientset
}

// NewClientset returns a new Clientset for the given config.
func NewClientset(appConfig *config.Config, k8sConfig *rest.Config) (*Clientset, error) {
	k8sClientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}
	logger := log.New(os.Stderr).With("component", "k8s")

	return &Clientset{
		appConfig,
		k8sConfig,
		logger,
		k8sClientset}, nil
}

// ExecPod issues an exec request to execute the given command to a particular
// pod.
func (c *Clientset) ExecPod(command []string) error {
	var (
		promNS         = c.config.GetString("namespace")
		promPod        = c.config.GetString("prometheus-pod")
		promContainer  = c.config.GetString("prometheus-container")
		requestTimeout = c.config.GetDuration("request-timeout")
		startTime      = c.config.GetTime("start-time")
		endTime        = c.config.GetTime("end-time")

		stdin  = true
		stdout = true
		stderr = true
		tty    = true
	)

	c.logger.Log("message", "exec into Prometheus pod",
		"namespace", promNS,
		"pod", promPod,
		"container", promContainer,
		"start-time", startTime,
		"end-time", endTime)

	execRequest := c.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(promNS).
		Name(promPod).
		SubResource("exec").
		Timeout(requestTimeout)

	execRequest.VersionedParams(&corev1.PodExecOptions{
		Container: promContainer,
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
