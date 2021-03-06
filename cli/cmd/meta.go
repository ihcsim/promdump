package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/ihcsim/promdump/pkg/config"
	"github.com/ihcsim/promdump/pkg/k8s"
	"github.com/spf13/cobra"
)

func initMetaCmd(rootCmd *cobra.Command) *cobra.Command {
	metaCmd := &cobra.Command{
		Use:   "meta -p POD [-n NAMESPACE] [-c CONTAINER] [-d DATA_DIR]",
		Short: "Shows the metadata of the Prometheus TSDB.",
		Example: `# show the metadata of all the data blocks in the Prometheus pod <pod> in
# namespace <ns>
kubectl promdump meta -p <pod> -n <ns>`,
		SilenceErrors: true, // let main() handles errors
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setMissingDefaults(cmd); err != nil {
				return fmt.Errorf("can't set missing defaults: %w", err)
			}

			if err := validateMetaOptions(cmd); err != nil {
				return fmt.Errorf("validation failed: %w", err)
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
	r := bytes.NewBuffer(promdumpBin)
	if err := uploadToContainer(r, config, clientset); err != nil {
		return err
	}
	defer func() {
		_ = clean(config, clientset)
	}()

	return printMeta(config, clientset)
}

func validateMetaOptions(cmd *cobra.Command) error {
	return nil
}

func printMeta(config *config.Config, clientset *k8s.Clientset) error {
	dataDir := config.GetString("data-dir")
	execCmd := []string{fmt.Sprintf("%s/promdump", dataDir), "-meta",
		"-data-dir", dataDir}
	if config.GetBool("debug") {
		execCmd = append(execCmd, "-debug")
	}

	return clientset.ExecPod(execCmd, os.Stdin, os.Stdout, os.Stderr, false)
}
