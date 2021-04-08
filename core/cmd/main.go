package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ihcsim/promdump/pkg/log"
	"github.com/ihcsim/promdump/pkg/tsdb"
	promtsdb "github.com/prometheus/prometheus/tsdb"
)

const (
	defaultLogLevel = "error"

	timeFormatFile = "2006-01-02-150405"
	timeFormatOut  = "2006-01-02 15:04:05"
)

var (
	logger             *log.Logger
	resultNoDataBlocks = "no data blocks found"
	targetDir          = os.TempDir()
)

func main() {
	var (
		defaultMaxTime = time.Now()
		defaultMinTime = defaultMaxTime.Add(-2 * time.Hour)

		dataDir  = flag.String("data-dir", "/data", "path to the Prometheus data directory")
		minTime  = flag.Int64("min-time", defaultMinTime.UnixNano(), "lower bound of the timestamp range (in nanoseconds)")
		maxTime  = flag.Int64("max-time", defaultMaxTime.UnixNano(), "upper bound of the timestamp range (in nanoseconds)")
		debug    = flag.Bool("debug", false, "run promdump in debug mode")
		showMeta = flag.Bool("meta", false, "retrieve the Promtheus TSDB metadata")
		help     = flag.Bool("help", false, "show usage")
	)
	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	logLevel := defaultLogLevel
	if *debug {
		logLevel = "debug"
	}
	logger = log.New(logLevel, os.Stderr)

	if err := validateTimestamp(*minTime, *maxTime); err != nil {
		exit(err)
	}

	tsdb := tsdb.New(*dataDir, logger)
	if *showMeta {
		meta, err := tsdb.Meta()
		if err != nil {
			exit(err)
		}

		if _, err := writeMeta(meta); err != nil {
			exit(err)
		}

		return
	}

	blocks, err := tsdb.Blocks(*minTime, *maxTime)
	if err != nil {
		exit(err)
	}

	nbr, err := writeBlocks(*dataDir, blocks, os.Stdout)
	if err != nil {
		exit(err)
	}

	_ = logger.Log("message", "operation completed", "numBytesRead", nbr)
}

func writeMeta(meta *tsdb.Meta) (int64, error) {
	if meta.Start.IsZero() && meta.End.IsZero() {
		return handleNoDataBlocks()
	}

	output := fmt.Sprintf(`Earliest time:          | %s
Latest time:            | %s
Total number of blocks  | %d
Total number of samples | %d
Total number of series  | %d
Total size              | %d
`,
		meta.Start.Format(timeFormatOut), meta.End.Format(timeFormatOut),
		meta.BlockCount, meta.TotalSamples, meta.TotalSeries, meta.TotalSize)

	buf := bytes.NewBuffer([]byte(output))
	return io.Copy(os.Stdout, buf)
}

func writeBlocks(dataDir string, blocks []*promtsdb.Block, w io.Writer) (int64, error) {
	if len(blocks) == 0 {
		return handleNoDataBlocks()
	}

	pipeReader, pipeWriter := io.Pipe()
	defer pipeReader.Close()

	go func() {
		defer pipeWriter.Close()
		if err := compressed(dataDir, blocks, pipeWriter); err != nil {
			_ = logger.Log("message", "error closing pipeWriter", "reason", err)
		}
	}()

	return io.Copy(w, pipeReader)
}

func compressed(dataDir string, blocks []*promtsdb.Block, writer *io.PipeWriter) error {
	var (
		buf = &bytes.Buffer{}
		tw  = tar.NewWriter(buf)
	)

	for _, block := range blocks {
		if err := filepath.Walk(block.Dir(), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			writeHeader := func(typeFlag byte) error {
				name := path[len(dataDir)+1:]
				header := &tar.Header{
					Name:     name,
					Mode:     int64(info.Mode()),
					ModTime:  info.ModTime(),
					Size:     info.Size(),
					Typeflag: typeFlag,
				}

				return tw.WriteHeader(header)
			}

			// if dir, only write header
			if info.IsDir() {
				return writeHeader(tar.TypeDir)
			}

			if err := writeHeader(tar.TypeReg); err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open data file: %w", err)
			}
			defer file.Close()

			data, err := ioutil.ReadAll(file)
			if err != nil {
				return fmt.Errorf("failed to read data file: %w", err)
			}

			if _, err := tw.Write(data); err != nil {
				return fmt.Errorf("failed to write compressed file: %w", err)
			}

			return nil
		}); err != nil {
			_ = logger.Log("errors", err)
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}

	now := time.Now()
	filename := fmt.Sprintf(filepath.Join(targetDir, "promdump-%s.tar.gz"), now.Format(timeFormatFile))

	gwriter := gzip.NewWriter(writer)
	defer gwriter.Close()

	gwriter.Header = gzip.Header{
		Name:    filename,
		ModTime: now,
		OS:      255,
	}

	if _, err := gwriter.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}

func validateTimestamp(minTime, maxTime int64) error {
	if minTime > maxTime {
		return fmt.Errorf("min-time (%d) cannot exceed max-time (%d)", minTime, maxTime)
	}

	now := time.Now().UnixNano()
	if minTime > now {
		return fmt.Errorf("min-time (%d) cannot exceed now (%d)", minTime, now)
	}

	if maxTime > now {
		return fmt.Errorf("max-time (%d) cannot exceed now (%d)", maxTime, now)
	}

	return nil
}

func handleNoDataBlocks() (int64, error) {
	buf := bytes.NewBuffer([]byte(resultNoDataBlocks + "\n"))
	return io.Copy(os.Stdout, buf)
}

func exit(err error) {
	_ = logger.Log("error", err)
	os.Exit(1)
}
