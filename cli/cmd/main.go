package main

import (
	"fmt"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	rootCmd, err := initRootCmd()
	if err != nil {
		exitWithErr(err)
	}

	_ = initMetaCmd(rootCmd)
	if _, err := initRestoreCmd(rootCmd); err != nil {
		exitWithErr(err)
	}

	if err := rootCmd.Execute(); err != nil {
		exitWithErr(err)
	}
}

func exitWithErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
