package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ihcsim/promdump/pkg/log"

	"github.com/ihcsim/promdump/pkg/config"
	"github.com/ihcsim/promdump/pkg/download"
	"github.com/ihcsim/promdump/pkg/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	k8scliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	downloadRemoteHost = "https://storage.googleapis.com/promdump"
	timeFormat         = "2006-01-02 15:04:05"
)

var (
	defaultEndTime        = time.Now()
	defaultStartTime      = defaultEndTime.Add(-1 * time.Hour)
	defaultLogLevel       = "error"
	defaultNamespace      = "default"
	defaultRequestTimeout = "10s"

	downloadRequestTimeout = time.Second * 10
	downloadLocalDir       = os.TempDir()

	appConfig      *config.Config
	clientset      *k8s.Clientset
	k8sConfigFlags *k8scliopts.ConfigFlags

	logger *log.Logger

	// Version is the version of the CLI, set during build time
	Version = "v0.1.0"
)

func initRootCmd() (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:   `promdump -p POD --start-time "yyyy-mm-dd hh:mm:ss" --end-time "yyyy-mm-dd hh:mm:ss" [-n NAMESPACE] [-c CONTAINER] [-d DATA_DIR]`,
		Short: "Creates a data dump containing Prometheus metric samples within a time range",
		Example: `# creates a data dump containing metric samples between
# 2021-01-01 00:00:00 and 2021-04-02 16:59:00, from the Prometheus <pod> in the
# <ns> namespace.
kubectl promdump -p <pod> -n <ns> --start-time "2021-01-01 00:00:00" --end-time "2021-04-02 16:59:00" > dump.tar.gz`,
		Long: `promdump creates a data dump containing Prometheus metric samples within a time range.

It is different from 'promtool tsdb dump' in such a way that its output can be
reused in another Prometheus instance. And unlike the Promethues TSDB snapshot
API, promdump doesn't require Prometheus to be started with the
--web.enable-admin-api option. Instead of dumping the entire TSDB, promdump
offers the flexibility to capture data that falls within a specific time range.

When run, the promdump CLI downloads the promdump tar.gz file from a public
Cloud Storage bucket to your local /tmp folder. The download will be skipped if
such a file already exists. The -f option can be used to force a re-download.

Then the CLI untar the archive file and upload the promdump binary to your
in-cluster Prometheus container, via the pod's exec subresource.

To create the data dump, promdump queries the Prometheus tsdb using the tsdb
package[1]. Data blocks that fall within the specified time range are gathered
and streamed to stdout, which can be redirected to a .tar.gz file on your local
file system. The promdump binary will be automatically removed from your
Prometheus container once the data dump is captured.

The 'restore' subcommand can be used to copy this .tar.gz file to another
Prometheus instance.

[1] https://pkg.go.dev/github.com/prometheus/prometheus/tsdb
		`,
		Version:       Version,
		SilenceErrors: true, // let main() handles errors
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setMissingDefaults(cmd); err != nil {
				return fmt.Errorf("can't set missing defaults: %w", err)
			}

			if err := validate(cmd); err != nil {
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
	rootCmd.PersistentFlags().StringP("container", "c", "prometheus", "Prometheus container name")
	rootCmd.PersistentFlags().StringP("data-dir", "d", "/data", "Prometheus data directory")
	rootCmd.PersistentFlags().Bool("debug", false, "run promdump in debug mode")
	rootCmd.PersistentFlags().BoolP("force", "f", false, "force the re-download of the promdump binary, which is saved to the local $TMP folder")
	rootCmd.Flags().String("start-time", defaultStartTime.Format(timeFormat), "start time (UTC) of the samples (yyyy-mm-dd hh:mm:ss)")
	rootCmd.Flags().String("end-time", defaultEndTime.Format(timeFormat), "end time (UTC) of the samples (yyyy-mm-dd hh:mm:ss")

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

func validate(cmd *cobra.Command) error {
	argStartTime, err := cmd.Flags().GetString("start-time")
	if err != nil {
		return err
	}

	argEndTime, err := cmd.Flags().GetString("end-time")
	if err != nil {
		return err
	}

	startTime, err := time.Parse(timeFormat, argStartTime)
	if err != nil {
		return err
	}

	endTime, err := time.Parse(timeFormat, argEndTime)
	if err != nil {
		return err
	}

	if startTime.After(endTime) {
		return fmt.Errorf("start time (%s) cannot be after end time (%s)", argStartTime, argEndTime)
	}

	now := time.Now()
	if startTime.After(now) {
		return fmt.Errorf("start time (%s) cannot be after now (%s)", argStartTime, now.Format(timeFormat))
	}

	if endTime.After(now) {
		return fmt.Errorf("end time (%s) cannot be after now (%s)", argEndTime, now.Format(timeFormat))
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
	bin, err := downloadBinary(cmd)
	if err != nil {
		return err
	}

	if err := uploadToContainer(bin, config, clientset); err != nil {
		return err
	}
	defer func() {
		_ = clean(config, clientset)
	}()

	return dumpSamples(config, clientset)
}

func downloadBinary(cmd *cobra.Command) (io.Reader, error) {
	var (
		remoteURI    = fmt.Sprintf("%s/promdump-%s.tar.gz", downloadRemoteHost, Version)
		remoteURISHA = fmt.Sprintf("%s/promdump-%s.sha256", downloadRemoteHost, Version)
	)

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return nil, err
	}

	download := download.New(downloadLocalDir, downloadRequestTimeout, logger)
	return download.Get(force, remoteURI, remoteURISHA)
}

func uploadToContainer(bin io.Reader, config *config.Config, clientset *k8s.Clientset) error {
	dataDir := config.GetString("data-dir")
	execCmd := []string{"tar", "-C", dataDir, "-xzvf", "-"}
	return clientset.ExecPod(execCmd, bin, ioutil.Discard, os.Stderr, false)
}

func dumpSamples(config *config.Config, clientset *k8s.Clientset) error {
	dataDir := config.GetString("data-dir")
	maxTime, err := time.Parse(timeFormat, config.GetString("end-time"))
	if err != nil {
		return err
	}
	minTime, err := time.Parse(timeFormat, config.GetString("start-time"))
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
