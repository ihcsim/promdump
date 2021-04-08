package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ihcsim/promdump/pkg/config"
	"github.com/ihcsim/promdump/pkg/k8s"
	"github.com/spf13/cobra"
)

func initRestoreCmd(rootCmd *cobra.Command) (*cobra.Command, error) {
	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "Restores samples dump to a Prometheus instance.",
		Example: `promdump restore -p prometheus-5c465dfc89-w72xp -n prometheus -d dump.tar.gz
`,
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

	data, err := ioutil.ReadAll(dumpFile)
	if err != nil {
		return fmt.Errorf("can't read sample dump file: %w", err)
	}

	return uploadToContainer(bytes.NewBuffer(data), config, clientset)
}
