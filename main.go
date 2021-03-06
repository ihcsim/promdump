package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/prometheus/tsdb"
)

var logger = initLogger()

func main() {
	var (
		defaultMaxTime = time.Now()
		defaultMinTime = defaultMaxTime.Add(-2 * time.Hour)

		dir     = flag.String("data-dir", "/prometheus", "path to the Prometheus data directory")
		maxTime = flag.Int64("max-time", defaultMaxTime.Unix(), "maximum timestamp of the data")
		minTime = flag.Int64("min-time", defaultMinTime.Unix(), "minimum timestamp of the data")
	)
	flag.Parse()

	logger.Log("dataDir", dir,
		"maxTime", time.Unix(*maxTime, 0),
		"minTime", time.Unix(*minTime, 0))

	db, err := tsdb.OpenDBReadOnly(*dir, logger)
	if err != nil {
		exit(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			panic(err)
		}
	}()

	blocks, err := db.Blocks()
	if err != nil {
		exit(err)
	}
	logger.Log("numBlocks", len(blocks))

	var (
		buf = &bytes.Buffer{}
		tw  = tar.NewWriter(buf)
	)

	for _, block := range blocks {
		b, ok := block.(*tsdb.Block)
		if !ok {
			continue
		}

		if b.MaxTime() <= *maxTime || b.MinTime() >= *minTime {
			blogger := log.With(logger, "block", b.Dir())
			err := filepath.Walk(b.Dir(), func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

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

				fnameIndex := strings.Index(path, b.Dir()) + len(b.Dir()) + 1
				flogger := log.With(blogger, "file", path[fnameIndex:])
				if err := writeHeader(tar.TypeReg); err != nil {
					return err
				}

				file, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open data file: %w", err)
				}
				defer file.Close()

				data := make([]byte, info.Size())
				numBytesRead, err := file.Read(data)
				if err != nil {
					return fmt.Errorf("failed to read data file: %w", err)
				}

				numBytesCompressed, err := tw.Write(data)
				if err != nil {
					return fmt.Errorf("failed to write compressed file: %w", err)
				}

				flogger.Log("numBytesRead", numBytesRead,
					"numBytesCompressed", numBytesCompressed)
				return nil
			})
			blogger.Log("errors", err)
		}
	}

	if err := tw.Close(); err != nil {
		exit(err)
	}

	now := time.Now()
	targetFilename := fmt.Sprintf("promdump-%s.tar.gz", now.Format("2006-01-02-150405"))
	targetFile, err := os.Create(targetFilename)
	if err != nil {
		exit(err)
	}

	gwriter := gzip.NewWriter(targetFile)
	defer gwriter.Close()

	gwriter.Header = gzip.Header{
		Name:    targetFilename,
		ModTime: now,
		OS:      255,
	}

	if _, err := gwriter.Write(buf.Bytes()); err != nil {
		exit(err)
	}
}

func initLogger() log.Logger {
	logger := log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger,
		"timestamp", log.DefaultTimestamp,
		"caller", log.DefaultCaller)
	return logger
}

func exit(err error) {
	logger.Log("error", err)
	os.Exit(1)
}
