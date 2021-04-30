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

	"github.com/go-kit/kit/log/level"
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
	logger                *log.Logger
	msgNoHeadBlock        = "No head block found"
	msgNoPersistentBlocks = "No persistent blocks found"
	targetDir             = os.TempDir()
)

func main() {
	var (
		defaultMaxTime = time.Now().UTC()
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

	tsdb, err := tsdb.New(*dataDir, logger)
	if err != nil {
		exit(err)
	}
	defer tsdb.Close()

	if *showMeta {
		headMeta, blockMeta, err := tsdb.Meta()
		if err != nil {
			exit(err)
		}

		if _, err := writeMeta(headMeta, blockMeta); err != nil {
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

	_ = level.Info(logger.Logger).Log("message", "operation completed", "numBytesRead", nbr)
}

func writeMeta(headMeta *tsdb.HeadMeta, blockMeta *tsdb.BlockMeta) (int64, error) {
	if headMeta.MinTime.IsZero() && headMeta.MaxTime.IsZero() {
		buf := bytes.NewBuffer([]byte(msgNoHeadBlock))
		return buf.WriteTo(os.Stdout)
	}

	head := fmt.Sprintf(`Head Block Metadata
------------------------
Minimum time (UTC): | %s
Maximum time (UTC): | %s
Number of series    | %d
`,
		headMeta.MinTime.Format(timeFormatOut),
		headMeta.MaxTime.Format(timeFormatOut),
		headMeta.NumSeries)

	buf := bytes.NewBuffer([]byte(head))
	if blockMeta.MinTime.IsZero() && blockMeta.MaxTime.IsZero() {
		if _, err := buf.Write([]byte("\n" + msgNoPersistentBlocks)); err != nil {
			return 0, err
		}
		return buf.WriteTo(os.Stdout)
	}

	blocks := fmt.Sprintf(`
Persistent Blocks Metadata
----------------------------
Minimum time (UTC):     | %s
Maximum time (UTC):     | %s
Total number of blocks  | %d
Total number of samples | %d
Total number of series  | %d
Total size              | %d
`,
		blockMeta.MinTime.Format(timeFormatOut),
		blockMeta.MaxTime.Format(timeFormatOut),
		blockMeta.BlockCount,
		blockMeta.NumSamples,
		blockMeta.NumSeries,
		blockMeta.Size)

	if _, err := buf.Write([]byte(blocks)); err != nil {
		return 0, err
	}

	return buf.WriteTo(os.Stdout)
}

func writeBlocks(dataDir string, blocks []*promtsdb.Block, w io.Writer) (int64, error) {
	if len(blocks) == 0 {
		buf := bytes.NewBuffer([]byte(msgNoPersistentBlocks))
		return buf.WriteTo(os.Stdout)
	}

	pipeReader, pipeWriter := io.Pipe()
	defer pipeReader.Close()

	go func() {
		defer pipeWriter.Close()
		if err := compressed(dataDir, blocks, pipeWriter); err != nil {
			_ = level.Error(logger.Logger).Log("message", "error closing pipeWriter", "reason", err)
		}
	}()

	return io.Copy(w, pipeReader)
}

func compressed(dataDir string, blocks []*promtsdb.Block, writer *io.PipeWriter) error {
	var (
		buf = &bytes.Buffer{}
		tw  = tar.NewWriter(buf)
	)

	dirs := []string{
		filepath.Join(dataDir, "chunks_head"),
		filepath.Join(dataDir, "wal"),
	}
	for _, block := range blocks {
		dirs = append(dirs, block.Dir())
	}

	// walk all the block directories
	for _, dir := range dirs {
		if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			var link string
			if info.Mode()&os.ModeSymlink == os.ModeSymlink {
				if link, err = os.Readlink(path); err != nil {
					return err
				}
			}

			header, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}

			header.Name = path[len(dataDir)+1:]
			if err = tw.WriteHeader(header); err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			data, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read data file: %w", err)
			}

			buf := bytes.NewBuffer(data)
			if _, err := io.Copy(tw, buf); err != nil {
				return fmt.Errorf("failed to write compressed file: %w", err)
			}

			return nil
		}); err != nil {
			_ = level.Error(logger.Logger).Log("errors", err)
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

func exit(err error) {
	_ = level.Error(logger.Logger).Log("error", err)
	os.Exit(1)
}
