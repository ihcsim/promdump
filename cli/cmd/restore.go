package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/ihcsim/promdump/pkg/config"
	"github.com/ihcsim/promdump/pkg/k8s"
	"github.com/spf13/cobra"
)

func initRestoreCmd(rootCmd *cobra.Command) (*cobra.Command, error) {
	restoreCmd := &cobra.Command{
		Use:   "restore -p POD [-n NAMESPACE] [-c CONTAINER] [-d DATA_DIR]",
		Short: "Restores data dump to a Prometheus instance.",
		Example: `# copy and restore the data dump in the dump.tar.gz file to the Prometheus
# <pod> in namespace <ns>.
kubectl promdump restore -p <pod> -n <ns> -t dump.tar.gz`,
		SilenceErrors: true, // let main() handles errors
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setMissingDefaults(cmd); err != nil {
				return fmt.Errorf("can't set missing defaults: %w", err)
			}

			if err := clientset.CanExec(); err != nil {
				return fmt.Errorf("exec operation denied: %w", err)
			}

			return runRestore(appConfig, clientset)
		},
	}

	restoreCmd.Flags().StringP("dump-file", "t", "", "path to the sample dump TAR file")
	if err := restoreCmd.MarkFlagRequired("dump-file"); err != nil {
		return nil, err
	}

	rootCmd.AddCommand(restoreCmd)
	return restoreCmd, nil
}

func runRestore(config *config.Config, clientset *k8s.Clientset) error {
	filename := config.GetString("dump-file")
	dumpFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("can't open dump file: %w", err)
	}

	data, err := io.ReadAll(dumpFile)
	if err != nil {
		return fmt.Errorf("can't read sample dump file: %w", err)
	}

	dataDir := config.GetString("data-dir")
	execCmd := []string{"sh", "-c", fmt.Sprintf("rm -rf %s/*", dataDir)}
	if err := clientset.ExecPod(execCmd, os.Stdin, os.Stdout, os.Stderr, false); err != nil {
		return err
	}

	return uploadToContainer(bytes.NewBuffer(data), config, clientset)
}
