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
func (t *Tsdb) Blocks(minTime, maxTime int64) ([]*tsdb.Block, error) {
	_ = t.logger.Log("message", "accessing tsdb", "datadir", t.dataDir)
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
			truncMinTime = b.MinTime() / int64(time.Microsecond)
			truncMaxTime = b.MaxTime() / int64(time.Microsecond)
			truncDir     = b.Dir()[len(t.dataDir)+2:]
			blockDir     = truncDir
		)
		if i := strings.Index(truncDir, "/"); i != -1 {
			blockDir = truncDir[:strings.Index(truncDir, "/")]
		}

		if truncMinTime >= minTime && truncMaxTime <= maxTime {
			if blockDir != current {
				current = blockDir
				t.logger.Log("message", "retrieving block",
					"name", b.Dir(),
					"minTime", time.Unix(truncMinTime, 0),
					"maxTime", time.Unix(truncMaxTime, 0),
				)
			}
			results = append(results, b)
		}
	}

	return results, nil
}
