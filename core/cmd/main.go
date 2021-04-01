package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
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

	_ = logger.Log("dataDir", dataDir,
		"maxTime", time.Unix(*maxTime, 0),
		"minTime", time.Unix(*minTime, 0))

	tsdb := tsdb.New(*dataDir, logger)
	blocks, err := tsdb.Blocks(*minTime, *maxTime)
	if err != nil {
		exit(err)
	}

	fileLocation, err := compressed(blocks)
	if err != nil {
		exit(err)
	}

	_ = logger.Log("message", "finished", "outputFile", fileLocation)
}

func compressed(blocks []*promtsdb.Block) (string, error) {
	var (
		buf = &bytes.Buffer{}
		tw  = tar.NewWriter(buf)
	)

	for _, block := range blocks {
		if err := filepath.Walk(block.Dir(), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			_ = logger.Log("message", "reading data block", "path", path)
			writeHeader := func(typeFlag byte) error {
				header := &tar.Header{
					Name:     path,
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

			numBytesCompressed, err := tw.Write(data)
			if err != nil {
				return fmt.Errorf("failed to write compressed file: %w", err)
			}

			_ = logger.Log("message", "read completed", "numBytesCompressed", numBytesCompressed)
			return nil
		}); err != nil {
			_ = logger.Log("errors", err)
		}
	}

	if err := tw.Close(); err != nil {
		return "", err
	}

	now := time.Now()
	filename := fmt.Sprintf(filepath.Join(targetDir, "promdump-%s.tar.gz"), now.Format(timeFormat))
	tarFile, err := os.Create(filename)
	if err != nil {
		return "", err
	}

	gwriter := gzip.NewWriter(tarFile)
	defer gwriter.Close()

	gwriter.Header = gzip.Header{
		Name:    filename,
		ModTime: now,
		OS:      255,
	}

	if _, err := gwriter.Write(buf.Bytes()); err != nil {
		return "", err
	}

	return filename, nil
}

func exit(err error) {
	_ = logger.Log("error", err)
	os.Exit(1)
}
