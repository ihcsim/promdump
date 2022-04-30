package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ihcsim/promdump/pkg/log"

	"github.com/ihcsim/promdump/pkg/config"
	"github.com/ihcsim/promdump/pkg/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	k8scliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	timeFormat = "2006-01-02 15:04:05"
)

var (
	defaultContainer      = "prometheus-server"
	defaultDataDir        = "/data"
	defaultDebugEnabled   = false
	defaultMaxTime        = time.Now()
	defaultLogLevel       = "error"
	defaultNamespace      = "default"
	defaultMinTime        = defaultMaxTime.Add(-1 * time.Hour)
	defaultRequestTimeout = "10s"

	appConfig      *config.Config
	clientset      *k8s.Clientset
	k8sConfigFlags *k8scliopts.ConfigFlags

	logger *log.Logger

	// Version is the version of the CLI, set during build time
	Version = "v0.1.0"
	Commit  = "unknown"
)

func initRootCmd() (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:   `promdump -p POD --min-time "yyyy-mm-dd hh:mm:ss" --max-time "yyyy-mm-dd hh:mm:ss" [-n NAMESPACE] [-c CONTAINER] [-d DATA_DIR]`,
		Short: "promdump dumps the head and persistent blocks of Prometheus",
		Example: `# dumps the head block and persistent blocks between
# 2021-01-01 00:00:00 and 2021-04-02 16:59:00, from the Prometheus <pod> in the
# <ns> namespace.
kubectl promdump -p <pod> -n <ns> --min-time "2021-01-01 00:00:00" --max-time "2021-04-02 16:59:00" > dump.tar.gz`,
		Long: `promdump dumps the head and persistent blocks of Prometheus. It supports
filtering the persistent blocks by time range.

promdump is a tool that can be used to dump Prometheus data blocks. It is
different from the 'promtool tsdb dump' command in such a way that its output
can be re-used in another Prometheus instance. And unlike the Promethues TSDB
'snapshot' API, promdump doesn't require Prometheus to be started with the
'--web.enable-admin-api' option. Instead of dumping the entire TSDB, promdump
offers the flexibility to filter persistent blocks by time range.

For more information on how promdump works, see
https://github.com/ihcsim/promdump.
		`,
		Version:       fmt.Sprintf("%s+%s", Version, Commit),
		SilenceErrors: true, // let main() handles errors
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setMissingDefaults(cmd); err != nil {
				return fmt.Errorf("can't set missing defaults: %w", err)
			}

			if err := validateRootOptions(cmd); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			if err := clientset.CanExec(); err != nil {
				return fmt.Errorf("exec operation denied: %w", err)
			}

			return run(cmd, appConfig, clientset)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			appConfig, err = config.New(cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to init viper config: %w", err)
			}

			k8sConfig, err := k8sConfig(k8sConfigFlags, cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to init k8s config: %w", err)
			}

			initLogger()
			clientset, err = k8s.NewClientset(appConfig, k8sConfig, logger)
			if err != nil {
				return fmt.Errorf("failed to init k8s client: %w", err)
			}

			return nil
		},
	}

	// add default k8s client flags
	k8sConfigFlags = k8scliopts.NewConfigFlags(true)
	k8sConfigFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.PersistentFlags().StringP("pod", "p", "", "Prometheus pod name")
	rootCmd.PersistentFlags().StringP("container", "c", defaultContainer, "Prometheus container name")
	rootCmd.PersistentFlags().StringP("data-dir", "d", defaultDataDir, "Prometheus data directory")
	rootCmd.PersistentFlags().Bool("debug", defaultDebugEnabled, "run promdump in debug mode")
	rootCmd.Flags().String("min-time", defaultMinTime.Format(timeFormat), "min time (UTC) of the samples (yyyy-mm-dd hh:mm:ss)")
	rootCmd.Flags().String("max-time", defaultMaxTime.Format(timeFormat), "max time (UTC) of the samples (yyyy-mm-dd hh:mm:ss)")

	rootCmd.Flags().SortFlags = false
	if err := rootCmd.MarkPersistentFlagRequired("pod"); err != nil {
		return nil, err
	}

	setPluginUsageTemplate(rootCmd)

	return rootCmd, nil
}

func setMissingDefaults(cmd *cobra.Command) error {
	ns, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}

	if ns == "" {
		if err := cmd.Flags().Set("namespace", defaultNamespace); err != nil {
			return err
		}
	}

	timeout, err := cmd.Flags().GetString("request-timeout")
	if err != nil {
		return err
	}

	if timeout == "0" {
		if err := cmd.Flags().Set("request-timeout", defaultRequestTimeout); err != nil {
			return err
		}
	}

	return nil
}

func validateRootOptions(cmd *cobra.Command) error {
	argMinTime, err := cmd.Flags().GetString("min-time")
	if err != nil {
		return err
	}

	argMaxTime, err := cmd.Flags().GetString("max-time")
	if err != nil {
		return err
	}

	minTime, err := time.Parse(timeFormat, argMinTime)
	if err != nil {
		return err
	}

	maxTime, err := time.Parse(timeFormat, argMaxTime)
	if err != nil {
		return err
	}

	if minTime.After(maxTime) {
		return fmt.Errorf("min time (%s) cannot be after max time (%s)", argMinTime, argMaxTime)
	}

	now := time.Now().UTC()
	if minTime.After(now) {
		return fmt.Errorf("min time (%s) cannot be after now (%s)", argMinTime, now.Format(timeFormat))
	}

	if maxTime.After(now) {
		return fmt.Errorf("max time (%s) cannot be after now (%s)", argMaxTime, now.Format(timeFormat))
	}

	return nil
}

func k8sConfig(k8sConfigFlags *k8scliopts.ConfigFlags, fs *pflag.FlagSet) (*rest.Config, error) {
	// read from CLI flags first
	// then if empty, load defaults from config loader
	currentContext, err := fs.GetString("context")
	if err != nil {
		return nil, err
	}

	configLoader := k8sConfigFlags.ToRawKubeConfigLoader()
	if currentContext == "" {
		rawConfig, err := configLoader.RawConfig()
		if err != nil {
			return nil, err
		}
		currentContext = rawConfig.CurrentContext
	}

	timeout, err := fs.GetString("request-timeout")
	if err != nil {
		return nil, err
	}

	if timeout == "" {
		clientConfig, err := configLoader.ClientConfig()
		if err != nil {
			return nil, err
		}
		timeout = clientConfig.Timeout.String()
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath:     configLoader.ConfigAccess().GetDefaultFilename(),
			WarnIfAllMissing: true,
		},
		&clientcmd.ConfigOverrides{
			CurrentContext: currentContext,
			Timeout:        timeout,
		}).ClientConfig()
}

func run(cmd *cobra.Command, config *config.Config, clientset *k8s.Clientset) error {
	r := bytes.NewReader(promdumpBin)
	if err := uploadToContainer(r, config, clientset); err != nil {
		return err
	}
	defer func() {
		_ = clean(config, clientset)
	}()

	return dumpSamples(config, clientset)
}

func uploadToContainer(bin io.Reader, config *config.Config, clientset *k8s.Clientset) error {
	dataDir := config.GetString("data-dir")
	execCmd := []string{"tar", "-C", dataDir, "-xzvf", "-"}
	return clientset.ExecPod(execCmd, bin, ioutil.Discard, os.Stderr, false)
}

func dumpSamples(config *config.Config, clientset *k8s.Clientset) error {
	dataDir := config.GetString("data-dir")
	maxTime, err := time.Parse(timeFormat, config.GetString("max-time"))
	if err != nil {
		return err
	}
	minTime, err := time.Parse(timeFormat, config.GetString("min-time"))
	if err != nil {
		return err
	}
	maxTimestamp := strconv.FormatInt(maxTime.UnixNano(), 10)
	minTimestamp := strconv.FormatInt(minTime.UnixNano(), 10)

	execCmd := []string{fmt.Sprintf("%s/promdump", dataDir),
		"-min-time", minTimestamp,
		"-max-time", maxTimestamp,
		"-data-dir", dataDir}
	if config.GetBool("debug") {
		execCmd = append(execCmd, "-debug")
	}

	return clientset.ExecPod(execCmd, os.Stdin, os.Stdout, os.Stderr, false)
}

func clean(config *config.Config, clientset *k8s.Clientset) error {
	dataDir := config.GetString("data-dir")
	execCmd := []string{"rm", "-f", fmt.Sprintf("%s/promdump", dataDir)}
	return clientset.ExecPod(execCmd, os.Stdin, os.Stdout, os.Stderr, false)
}

func initLogger() {
	logLevel := defaultLogLevel
	if appConfig.GetBool("debug") {
		logLevel = "debug"
	}

	logger = log.New(logLevel, os.Stderr)
}

func setPluginUsageTemplate(cmd *cobra.Command) {
	defaultTmpl := cmd.UsageTemplate()
	newTmpl := strings.ReplaceAll(defaultTmpl, "{{.CommandPath}}", "kubectl {{.CommandPath}}")
	newTmpl = strings.ReplaceAll(newTmpl, "{{.UseLine}}", "kubectl {{.UseLine}}")
	cmd.SetUsageTemplate(newTmpl)
}
