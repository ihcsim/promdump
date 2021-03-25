package main

import "github.com/spf13/cobra"

func initRestoreCmd(rootCmd *cobra.Command) *cobra.Command {
	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	rootCmd.AddCommand(restoreCmd)
	return restoreCmd
}
