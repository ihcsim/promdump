package main

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ihcsim/promdump/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	cmdFixture    *cobra.Command
	configFixture *config.Config
)

func TestConfigFromFlagset(t *testing.T) {
	if err := initFixtures(); err != nil {
		t.Fatal("unexpected error: ", err)
	}

	if err := cmdFixture.Execute(); err != nil {
		t.Fatal("unexpected error: ", err)
	}

	args := buildArgsFromFlags(cmdFixture, t)
	cmdFixture.SetArgs(args)

	if err := cmdFixture.Execute(); err != nil {
		t.Fatal("unexpected error: ", err)
	}

	assertConfig(cmdFixture, configFixture, t)
}

func initFixtures() error {
	var err error
	cmdFixture, err = initRootCmd()
	if err != nil {
		return err
	}

	cmdFixture.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		configFixture, err = config.FromFlagSet(cmd.Flags())
		if err != nil {
			return fmt.Errorf("failed to init viper config: %w", err)
		}

		return nil
	}

	cmdFixture.RunE = func(cmd *cobra.Command, args []string) error {
		return nil // omit irrelevant streaming function
	}

	// preset required fields with no default value
	if err := cmdFixture.PersistentFlags().Set("prometheus-pod", "test-pod"); err != nil {
		return err
	}

	cmdFixture.SetOutput(ioutil.Discard)
	return nil
}

func buildArgsFromFlags(cmd *cobra.Command, t *testing.T) []string {
	// construct the CLI arguments
	var args []string
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if skipFlag(f) {
			return
		}

		args = append(args, testArgs(f, t)...)
	})

	return args
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

func assertConfig(cmd *cobra.Command, appConfig *config.Config, t *testing.T) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if skipFlag(f) {
			return
		}

		t.Run(f.Name, func(t *testing.T) {
			expected := expectedValue(f, t)
			if actual := appConfig.Get(f.Name); !reflect.DeepEqual(expected, actual) {
				t.Errorf("mismatch config: %s. expected: %v (%T), actual: %v (%T)", f.Name, expected, expected, actual, actual)
			}
		})
	})
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
