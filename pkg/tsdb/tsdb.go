package tsdb

import (
	"strings"
	"time"

	"github.com/ihcsim/promdump/pkg/log"
	"github.com/prometheus/prometheus/tsdb"
)

// Tsdb knows how to access a Prometheus tsdb.
type Tsdb struct {
	dataDir string
	logger  *log.Logger
}

// New returns a new instance of Tsdb.
func New(dataDir string, logger *log.Logger) *Tsdb {
	return &Tsdb{dataDir, logger}
}

// Blocks looks for data blocks that fall within the provided time range, in the
// data directory.
func (t *Tsdb) Blocks(minTimeNano, maxTimeNano int64) ([]*tsdb.Block, error) {
	var (
		startTime = time.Unix(0, minTimeNano)
		endTime   = time.Unix(0, maxTimeNano)
	)
	_ = t.logger.Log("message", "accessing tsdb",
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
		_ = t.logger.Log("message", "closing connection to tsdb", "numBlocksFound", len(results))
		_ = db.Close()
	}()

	var current string
	for _, block := range blocks {
		b, ok := block.(*tsdb.Block)
		if !ok {
			continue
		}

		var (
			blockStartTime = time.Unix(0, nanoseconds(b.MinTime()))
			blockEndTime   = time.Unix(0, nanoseconds(b.MaxTime()))
			truncDir       = b.Dir()[len(t.dataDir)+1:]
			blockDir       = truncDir
		)
		if i := strings.Index(truncDir, "/"); i != -1 {
			blockDir = truncDir[:strings.Index(truncDir, "/")]
		}

		_ = t.logger.Log("message", "checking block",
			"path", blockDir,
			"blockStartTime", blockStartTime,
			"blockEndTime", blockEndTime,
		)

		if (blockStartTime.After(startTime) || blockStartTime.Equal(startTime)) &&
			(blockEndTime.Before(endTime) || blockEndTime.Equal(endTime)) {
			if blockDir != current {
				current = blockDir
				_ = t.logger.Log("message", "adding block", "path", blockDir)
			}
			results = append(results, b)
		} else {
			_ = t.logger.Log("message", "skipping block", "path", blockDir)
		}
	}

	return results, nil
}

func nanoseconds(milliseconds int64) int64 {
	return milliseconds * 1000000
}
