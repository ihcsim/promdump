package tsdb

import (
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
func (t *Tsdb) Blocks(maxTime, minTime int64) ([]*tsdb.Block, error) {
	_ = t.logger.Log("message", "accessing tsdb", "datadir", t.dataDir)
	db, err := tsdb.OpenDBReadOnly(t.dataDir, t.logger.Logger)
	if err != nil {
		return nil, err
	}

	blocks, err := db.Blocks()
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = t.logger.Log("message", "closing connection to tsdb", "numBlocksFound", len(blocks))
		_ = db.Close()
	}()

	var results []*tsdb.Block
	for _, block := range blocks {
		b, ok := block.(*tsdb.Block)
		if !ok {
			continue
		}

		if b.MaxTime() <= maxTime || b.MinTime() >= minTime {
			results = append(results, b)
		}
	}

	return results, nil
}
