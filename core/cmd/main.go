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

const timeFormat = "2006-01-02-150405"

var (
	logger    = log.New(os.Stderr)
	targetDir = os.TempDir()
)

func main() {
	var (
		defaultMaxTime = time.Now()
		defaultMinTime = defaultMaxTime.Add(-2 * time.Hour)

		dataDir = flag.String("data-dir", "/prometheus", "path to the Prometheus data directory")
		maxTime = flag.Int64("max-time", defaultMaxTime.Unix(), "maximum timestamp of the data")
		minTime = flag.Int64("min-time", defaultMinTime.Unix(), "minimum timestamp of the data")
		help    = flag.Bool("help", false, "show usage")
	)
	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	_ = logger.Log("message", "starting promdump",
		"dataDir", dataDir,
		"maxTime", time.Unix(*maxTime, 0),
		"minTime", time.Unix(*minTime, 0))

	tsdb := tsdb.New(*dataDir, logger)
	blocks, err := tsdb.Blocks(*minTime, *maxTime)
	if err != nil {
		exit(err)
	}

	pipeReader, pipeWriter := io.Pipe()
	defer pipeReader.Close()

	go func() {
		defer pipeWriter.Close()
		if err := compressed(*dataDir, blocks, pipeWriter); err != nil {
			logger.Log("message", "error closing pipeWriter", "reason", err)
		}
	}()

	nbr, err := io.Copy(os.Stdout, pipeReader)
	if err != nil {
		exit(err)
	}
	logger.Log("message", "operation completed", "numBytesRead", nbr)
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
				name := path[len(dataDir)+2:]
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
	filename := fmt.Sprintf(filepath.Join(targetDir, "promdump-%s.tar.gz"), now.Format(timeFormat))

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

func exit(err error) {
	_ = logger.Log("error", err)
	os.Exit(1)
}
