package main

import (
	"fmt"
	"os"
)

func main() {
	rootCmd, err := initRootCmd()
	if err != nil {
		exitWithErr(err)
	}
	_ = initRestoreCmd(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		exitWithErr(err)
	}
}

func exitWithErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
