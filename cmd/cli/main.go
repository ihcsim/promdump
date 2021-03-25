package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const timeFormat = "2006-01-02 15:04:05"

var (
	defaultKubeConfig = filepath.Join("~", ".kube", "config")
	defaultStartTime  = time.Now()
	defaultEndTime    = defaultStartTime.Add(-1 * time.Hour)
)

func main() {
	rootCmd, err := initRootCmd()
	if err != nil {
		exitWithErr(err)
	}
	initRestoreCmd(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		exitWithErr(err)
	}
}

func exitWithErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
