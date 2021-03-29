package main

import (
	"fmt"
	"time"

	"github.com/ihcsim/promdump/pkg/config"
	"github.com/ihcsim/promdump/pkg/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	k8scliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const timeFormat = "2006-01-02 15:04:05"

var (
	defaultStartTime = time.Now()
	defaultEndTime   = defaultStartTime.Add(-1 * time.Hour)

	// Version is the version of the CLI, set during build time
	Version = "unknown"
)

func initRootCmd() (*cobra.Command, error) {
	var (
		clientset      *k8s.Clientset
		k8sConfigFlags *k8scliopts.ConfigFlags
	)

	rootCmd := &cobra.Command{
		Use:           "promdump",
		Example:       "",
		Short:         "A tool to dump Prometheus tsdb samples within a time range",
		Long:          ``,
		Version:       Version,
		SilenceErrors: true, // let main() handles errors
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate(cmd); err != nil {
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

			k8sConfig, err := k8sConfig(k8sConfigFlags, cmd.Flags())
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
	k8sConfigFlags = k8scliopts.NewConfigFlags(true)
	k8sConfigFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.PersistentFlags().StringP("pod", "p", "", "targeted Prometheus pod name")
	rootCmd.PersistentFlags().StringP("container", "c", "prometheus", "targeted Prometheus container name")
	rootCmd.Flags().String("start-time", defaultStartTime.Format(timeFormat), "start time (UTC) of the samples (yyyy-mm-dd hh:mm:ss)")
	rootCmd.Flags().String("end-time", defaultEndTime.Format(timeFormat), "end time (UTC) of the samples (yyyy-mm-dd hh:mm:ss")

	rootCmd.Flags().SortFlags = false
	if err := rootCmd.MarkPersistentFlagRequired("pod"); err != nil {
		return nil, err
	}

	return rootCmd, nil
}

func validate(cmd *cobra.Command) error {
	const errPrefix = "flag validation failed"
	argStartTime, err := cmd.Flags().GetString("start-time")
	if err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}

	argEndTime, err := cmd.Flags().GetString("end-time")
	if err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}

	startTime, err := time.Parse(timeFormat, argStartTime)
	if err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}

	endTime, err := time.Parse(timeFormat, argEndTime)
	if err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}

	if startTime.After(endTime) {
		return fmt.Errorf("invalid time range. start time (%s) must occur before end time (%s).", argStartTime, argEndTime)
	}

	return nil
}

func k8sConfig(k8sConfigFlags *k8scliopts.ConfigFlags, fs *pflag.FlagSet) (*rest.Config, error) {
	// read from CLI flags first
	// then if empty, load defaults from config loader
	currentContext, err := fs.GetString("context")
	if err != nil {
		return nil, err
	}

	timeout, err := fs.GetString("request-timeout")
	if err != nil {
		return nil, err
	}

	configLoader := k8sConfigFlags.ToRawKubeConfigLoader()
	if currentContext == "" {
		rawConfig, err := configLoader.RawConfig()
		if err != nil {
			return nil, err
		}
		currentContext = rawConfig.CurrentContext
	}

	if timeout == "" {
		clientConfig, err := configLoader.ClientConfig()
		if err != nil {
			return nil, err
		}
		timeout = clientConfig.Timeout.String()
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath:     configLoader.ConfigAccess().GetDefaultFilename(),
			WarnIfAllMissing: true,
		},
		&clientcmd.ConfigOverrides{
			CurrentContext: currentContext,
			Timeout:        timeout,
		}).ClientConfig()
}
