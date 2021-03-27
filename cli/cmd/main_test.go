package main

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestInitConfig(t *testing.T) {
	rootCmd, err := initRootCmd()
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}
	rootCmd.SetOutput(ioutil.Discard)
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return initConfig(nil, rootCmd.Flags())
	}
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// omit running the streaming function
		return nil
	}

	// parse flags
	if err := rootCmd.Execute(); err != nil {
		t.Fatal("unexpected error: ", err)
	}

	var args []string
	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		if skipFlag(f) {
			return
		}

		args = append(args, testArgs(f, t)...)
	})
	rootCmd.SetArgs(args)

	// execute command with new args
	if err := rootCmd.Execute(); err != nil {
		t.Fatal("unexpected error: ", err)
	}

	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		if skipFlag(f) {
			return
		}

		t.Run(f.Name, func(t *testing.T) {
			expected := expectedValue(f, t)
			if actual := _config.Get(f.Name); !reflect.DeepEqual(expected, actual) {
				t.Errorf("mismatch config: %s. expected: %v (%T), actual: %v (%T)", f.Name, expected, expected, actual, actual)
			}
		})
	})
}

func skipFlag(f *pflag.Flag) bool {
	return f.Name == "help" || f.Name == "version"
}

// build the arguments to be passed to the command for testing.
// returns a slice in the form of {"--flag", "test-flag"}.
func testArgs(f *pflag.Flag, t *testing.T) []string {
	args := []string{fmt.Sprintf("--%s", f.Name)}

	switch f.Value.Type() {
	case "bool":
		return args
	case "string":
		args = append(args, fmt.Sprintf("test-%s", f.Name))
	case "stringArray":
		args = append(args, fmt.Sprintf("test-%s-00", f.Name),
			fmt.Sprintf("--%s", f.Name),
			fmt.Sprintf("test-%s-01", f.Name))
	default:
		t.Fatalf("unsupported type: %s (flag: %s)", f.Value.Type(), f.Name)
	}

	return args
}

func expectedValue(f *pflag.Flag, t *testing.T) interface{} {
	switch f.Value.Type() {
	case "bool":
		return true
	case "string":
		return fmt.Sprintf("test-%s", f.Name)
	case "stringArray":
		return fmt.Sprintf("[test-%s-00,test-%s-01]", f.Name, f.Name)
	default:
		t.Fatalf("unsupported type: %s (%s)", f.Value.Type(), f.Name)
	}

	return nil
}
