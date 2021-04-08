package tsdb

import (
	"strings"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/ihcsim/promdump/pkg/log"
	"github.com/prometheus/prometheus/tsdb"
)

// Tsdb knows how to access a Prometheus tsdb.
type Tsdb struct {
	dataDir string
	logger  *log.Logger
}

// Meta contains metadata for a TSDB instance.
type Meta struct {
	BlockCount int

	Start time.Time
	End   time.Time

	TotalSamples uint64
	TotalSeries  uint64
	TotalSize    int64
}

// New returns a new instance of Tsdb.
func New(dataDir string, logger *log.Logger) *Tsdb {
	return &Tsdb{dataDir, logger}
}

// Blocks looks for data blocks that fall within the provided time range, in the
// data directory.
func (t *Tsdb) Blocks(minTimeNano, maxTimeNano int64) ([]*tsdb.Block, error) {
	var (
		startTime = time.Unix(0, minTimeNano).UTC()
		endTime   = time.Unix(0, maxTimeNano).UTC()
	)
	_ = level.Debug(t.logger).Log("message", "accessing tsdb",
		"datadir", t.dataDir,
		"startTime", startTime,
		"endTime", endTime)

	db, err := tsdb.OpenDBReadOnly(t.dataDir, t.logger.Logger)
	if err != nil {
		return nil, err
	}

	blocks, err := db.Blocks()
	if err != nil {
		return nil, err
	}

	var results []*tsdb.Block
	defer func() {
		_ = level.Debug(t.logger).Log("message", "closing connection to tsdb", "numBlocksFound", len(results))
		_ = db.Close()
	}()

	var current string
	for _, block := range blocks {
		b, ok := block.(*tsdb.Block)
		if !ok {
			continue
		}

		var (
			blockStartTime = time.Unix(0, nanoseconds(b.MinTime())).UTC()
			blockEndTime   = time.Unix(0, nanoseconds(b.MaxTime())).UTC()
			truncDir       = b.Dir()[len(t.dataDir)+1:]
			blockDir       = truncDir
		)
		if i := strings.Index(truncDir, "/"); i != -1 {
			blockDir = truncDir[:strings.Index(truncDir, "/")]
		}

		_ = level.Debug(t.logger).Log("message", "checking block",
			"path", blockDir,
			"blockStartTime (utc)", blockStartTime,
			"blockEndTime (utc)", blockEndTime,
		)

		if startTime.Equal(blockStartTime) || endTime.Equal(blockEndTime) ||
			(endTime.After(blockStartTime) && endTime.Before(blockEndTime)) ||
			(startTime.After(blockStartTime) && startTime.Before(blockEndTime)) ||
			(startTime.Before(blockStartTime) && endTime.After(blockEndTime)) {

			if blockDir != current {
				current = blockDir
				_ = level.Debug(t.logger).Log("message", "adding block", "path", blockDir)
			}
			results = append(results, b)
		} else {
			_ = level.Debug(t.logger).Log("message", "skipping block", "path", blockDir)
		}
	}

	return results, nil
}

// Meta returns metadata of the TSDB instance.
func (t *Tsdb) Meta() (*Meta, error) {
	_ = level.Debug(t.logger).Log("message", "retrieving tsdb metadata", "datadir", t.dataDir)
	db, err := tsdb.OpenDBReadOnly(t.dataDir, t.logger.Logger)
	if err != nil {
		return nil, err
	}

	blocks, err := db.Blocks()
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = level.Debug(t.logger).Log("message", "closing connection to tsdb")
		_ = db.Close()
	}()

	var (
		earliest time.Time
		latest   time.Time

		totalSize    int64
		totalSamples uint64
		totalSeries  uint64
	)
	for _, block := range blocks {
		b, ok := block.(*tsdb.Block)
		if !ok {
			continue
		}

		var (
			blockStartTime = time.Unix(0, nanoseconds(b.MinTime())).UTC()
			blockEndTime   = time.Unix(0, nanoseconds(b.MaxTime())).UTC()
			truncDir       = b.Dir()[len(t.dataDir)+1:]
			blockDir       = truncDir
		)
		if i := strings.Index(truncDir, "/"); i != -1 {
			blockDir = truncDir[:strings.Index(truncDir, "/")]
		}

		_ = level.Debug(t.logger).Log("message", "checking block",
			"path", blockDir,
			"blockStartTime", blockStartTime,
			"blockEndTime", blockEndTime,
		)

		totalSize += b.Size()
		totalSamples += b.Meta().Stats.NumSamples
		totalSeries += b.Meta().Stats.NumSeries

		if blockStartTime.Before(earliest) || earliest.IsZero() {
			earliest = blockStartTime
		}

		if blockEndTime.After(latest) || latest.IsZero() {
			latest = blockEndTime
		}
	}

	return &Meta{
		BlockCount:   len(blocks),
		Start:        earliest,
		End:          latest,
		TotalSamples: totalSamples,
		TotalSize:    totalSize,
		TotalSeries:  totalSeries,
	}, nil
}

func nanoseconds(milliseconds int64) int64 {
	return milliseconds * 1000000
}
