package main

import (
	"fmt"
	"os"

	"github.com/ihcsim/promdump/pkg/config"
	"github.com/ihcsim/promdump/pkg/k8s"
	"github.com/spf13/cobra"
)

func initMetaCmd(rootCmd *cobra.Command) *cobra.Command {
	metaCmd := &cobra.Command{
		Use:           "meta",
		Short:         "Show the metadata of the TSDB.",
		Example:       `promdump meta -p prometheus-5c465dfc89-w72xp -n prometheus`,
		SilenceErrors: true, // let main() handles errors
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setMissingDefaults(cmd); err != nil {
				return fmt.Errorf("can't set missing defaults: %w", err)
			}

			if err := clientset.CanExec(); err != nil {
				return fmt.Errorf("exec operation denied: %w", err)
			}

			return runMeta(cmd, appConfig, clientset)
		},
	}

	rootCmd.AddCommand(metaCmd)
	return metaCmd
}

func runMeta(cmd *cobra.Command, config *config.Config, clientset *k8s.Clientset) error {
	bin, err := downloadBinary(cmd, config)
	if err != nil {
		return err
	}

	if err := uploadToContainer(bin, config, clientset); err != nil {
		return err
	}
	defer func() {
		_ = clean(config, clientset)
	}()

	return printMeta(config, clientset)
}

func printMeta(config *config.Config, clientset *k8s.Clientset) error {
	dataDir := config.GetString("prometheus.dataDir")
	execCmd := []string{fmt.Sprintf("%s/promdump", dataDir), "-meta"}
	return clientset.ExecPod(execCmd, os.Stdin, os.Stdout, os.Stderr, false)
}
