package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	k8scliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

var Version = "unknown"

func initRootCmd() (*cobra.Command, error) {
	var (
		k8s            *kubernetes.Clientset
		k8sConfigFlags *k8scliopts.ConfigFlags
	)

	rootCmd := &cobra.Command{
		Use:     "promdump",
		Example: "",
		Short:   "A tool to dump Prometheus tsdb samples within a time range",
		Long:    ``,
		Version: Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate(args); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			return execToPod(k8s)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			k8sConfig, err := k8sConfigFlags.ToRawKubeConfigLoader().ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to init k8s config: %w", err)
			}

			k8s, err = kubernetes.NewForConfig(k8sConfig)
			if err != nil {
				return fmt.Errorf("failed to init k8s client: %w", err)
			}

			if err := initConfig(k8sConfig, cmd.Flags()); err != nil {
				return fmt.Errorf("failed to init viper config: %w", err)
			}

			return nil
		},
	}

	// add default k8s client flags
	k8sConfigFlags = k8scliopts.NewConfigFlags(false)
	k8sConfigFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.PersistentFlags().StringP("prometheus-pod", "p", "", "targeted Prometheus pod name")
	rootCmd.Flags().String("start-time", defaultStartTime.Format(timeFormat), "start time (UTC) of the samples (yyyy-mm-dd hh:mm:ss)")
	rootCmd.Flags().String("end-time", defaultEndTime.Format(timeFormat), "end time (UTC) of the samples (yyyy-mm-dd hh:mm:ss")

	rootCmd.MarkFlagRequired("pod")
	rootCmd.Flags().SortFlags = false

	return rootCmd, nil
}

func validate(args []string) error {
	return nil
}

func execToPod(k8s *kubernetes.Clientset) error {
	promNS := _config.GetString("namespace")
	promPod := _config.GetString("pod")
	promContainer := _config.GetString("container")
	execRequest := k8s.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(promNS).
		Namespace(promPod).
		SubResource("exec")

	stdin := true
	stdout := true
	stderr := true
	tty := true
	execRequest.VersionedParams(&corev1.PodExecOptions{
		Container: promContainer,
		Command:   []string{"ls"},
		Stdin:     stdin,
		Stdout:    stdout,
		Stderr:    stderr,
		TTY:       tty,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(_config.k8s, "POST", execRequest.URL())
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
