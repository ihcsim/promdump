package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/ihcsim/promdump/pkg/config"
	"github.com/ihcsim/promdump/pkg/k8s"
	"github.com/spf13/cobra"
	k8scliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

const timeFormat = "2006-01-02 15:04:05"

var (
	defaultKubeConfig = filepath.Join("~", ".kube", "config")
	defaultStartTime  = time.Now()
	defaultEndTime    = defaultStartTime.Add(-1 * time.Hour)

	// Version is the version of the CLI, set during build time
	Version = "unknown"
)

func initRootCmd() (*cobra.Command, error) {
	var (
		clientset      *k8s.Clientset
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

			execCmd := []string{"/bin/sh"}
			return clientset.ExecPod(execCmd)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			appConfig, err := config.FromFlagSet(cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to init viper config: %w", err)
			}

			k8sConfig, err := k8sConfigFlags.ToRawKubeConfigLoader().ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to init k8s config: %w", err)
			}

			clientset, err = k8s.NewClientset(appConfig, k8sConfig)
			if err != nil {
				return fmt.Errorf("failed to init k8s client: %w", err)
			}

			return nil
		},
	}

	// add default k8s client flags
	k8sConfigFlags = k8scliopts.NewConfigFlags(false)
	k8sConfigFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.PersistentFlags().StringP("prometheus-pod", "p", "", "targeted Prometheus pod name")
	rootCmd.PersistentFlags().StringP("prometheus-container", "c", "prometheus", "targeted Prometheus container name")
	rootCmd.Flags().String("start-time", defaultStartTime.Format(timeFormat), "start time (UTC) of the samples (yyyy-mm-dd hh:mm:ss)")
	rootCmd.Flags().String("end-time", defaultEndTime.Format(timeFormat), "end time (UTC) of the samples (yyyy-mm-dd hh:mm:ss")

	rootCmd.MarkFlagRequired("pod")
	rootCmd.Flags().SortFlags = false

	return rootCmd, nil
}

func validate(args []string) error {
	return nil
}
