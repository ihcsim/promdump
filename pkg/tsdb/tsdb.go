package tsdb

import (
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/ihcsim/promdump/pkg/log"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/prometheus/prometheus/tsdb/chunkenc"
	"github.com/prometheus/prometheus/tsdb/wal"
)

// Tsdb knows how to access a Prometheus tsdb.
type Tsdb struct {
	dataDir string
	db      *tsdb.DBReadOnly
	logger  *log.Logger
}

// HeadMeta contains metadata of the head block.
type HeadMeta struct {
	*Meta
	NumChunks uint64
}

// BlockMeta contains aggregated metadata of all the persistent blocks.
type BlockMeta struct {
	*Meta
	BlockCount int
}

// Meta contains metadata for a TSDB instance.
type Meta struct {
	MaxTime    time.Time
	MinTime    time.Time
	NumSamples uint64
	NumSeries  uint64
	Size       int64
}

// New returns a new instance of Tsdb.
func New(dataDir string, logger *log.Logger) (*Tsdb, error) {
	db, err := tsdb.OpenDBReadOnly(dataDir, logger)
	if err != nil {
		return nil, err
	}
	return &Tsdb{dataDir, db, logger}, nil
}

// Close closes the underlying database connection.
func (t *Tsdb) Close() error {
	_ = level.Debug(t.logger).Log("message", "closing connection to tsdb")
	return t.db.Close()
}

// Blocks looks for data blocks that fall within the provided time range, in the
// data directory.
func (t *Tsdb) Blocks(minTimeNano, maxTimeNano int64) ([]*tsdb.Block, error) {
	var (
		minTime = time.Unix(0, minTimeNano).UTC()
		maxTime = time.Unix(0, maxTimeNano).UTC()
	)
	_ = level.Debug(t.logger).Log("message", "accessing tsdb",
		"datadir", t.dataDir,
		"minTime", minTime,
		"maxTime", maxTime)

	var (
		results []*tsdb.Block
		current string
	)

	blocks, err := t.db.Blocks()
	if err != nil {
		return nil, err
	}

	for _, block := range blocks {
		b, ok := block.(*tsdb.Block)
		if !ok {
			continue
		}

		var (
			blMinTime = time.Unix(0, nanoseconds(b.MinTime())).UTC()
			blMaxTime = time.Unix(0, nanoseconds(b.MaxTime())).UTC()
			truncDir  = b.Dir()[len(t.dataDir)+1:]
			blockDir  = truncDir
		)
		if i := strings.Index(truncDir, "/"); i != -1 {
			blockDir = truncDir[:strings.Index(truncDir, "/")]
		}

		_ = level.Debug(t.logger).Log("message", "checking block",
			"path", blockDir,
			"minTime (utc)", blMinTime,
			"maxTime (utc)", blMaxTime,
		)

		if minTime.Equal(blMinTime) || maxTime.Equal(blMaxTime) ||
			(maxTime.After(blMinTime) && maxTime.Before(blMaxTime)) ||
			(minTime.After(blMinTime) && minTime.Before(blMaxTime)) ||
			(minTime.Before(blMinTime) && maxTime.After(blMaxTime)) {

			if blockDir != current {
				current = blockDir
				_ = level.Debug(t.logger).Log("message", "adding block", "path", blockDir)
			}
			results = append(results, b)
		} else {
			_ = level.Debug(t.logger).Log("message", "skipping block", "path", blockDir)
		}
	}

	_ = level.Debug(t.logger).Log("message", "finish parsing persistent blocks", "numBlocksFound", len(results))
	return results, nil
}

// BlockMeta returns metadata of the TSDB persistent blocks.
func (t *Tsdb) Meta() (*HeadMeta, *BlockMeta, error) {
	_ = level.Debug(t.logger).Log("message", "retrieving tsdb metadata", "datadir", t.dataDir)
	headMeta, err := t.headMeta()
	if err != nil {
		return nil, nil, err
	}

	blockMeta, err := t.blockMeta()
	if err != nil {
		return nil, nil, err
	}

	return headMeta, blockMeta, nil
}

// headMeta() is based on the implementation of tsdb.FlushWAL() at
// https://github.com/prometheus/prometheus/blob/80545bfb2eb8f9deeedc442130f7c4dc34525d8d/tsdb/db.go#L334
func (t *Tsdb) headMeta() (*HeadMeta, error) {
	dir := filepath.Join(t.dataDir, "wal")
	_ = level.Debug(t.logger).Log("message", "retrieving head block metadata", "datadir", dir)
	wal, err := wal.Open(t.logger, dir)
	if err != nil {
		return nil, err
	}
	defer wal.Close()

	head, err := tsdb.NewHead(nil, t.logger, wal,
		tsdb.DefaultBlockDuration,
		"",
		chunkenc.NewPool(),
		tsdb.DefaultStripeSize,
		nil)
	if err != nil {
		return nil, err
	}

	blocks, err := t.db.Blocks()
	if err != nil {
		return nil, err
	}

	minValidTime := int64(math.MinInt64)
	if len(blocks) > 0 {
		minValidTime = blocks[len(blocks)-1].Meta().MaxTime
	}
	if err := head.Init(minValidTime); err != nil {
		return nil, err
	}

	// NumSamples and NumChunks are not populated by default. See
	// https://github.com/prometheus/prometheus/blob/80545bfb2eb8f9deeedc442130f7c4dc34525d8d/tsdb/head.go#L1600
	return &HeadMeta{
		Meta: &Meta{
			MaxTime:   time.Unix(0, nanoseconds(head.MaxTime())).UTC(),
			MinTime:   time.Unix(0, nanoseconds(head.MinTime())).UTC(),
			NumSeries: head.Meta().Stats.NumSeries,
		},
	}, nil
}

func (t *Tsdb) blockMeta() (*BlockMeta, error) {
	_ = level.Debug(t.logger).Log("message", "retrieving persistent blocks metadata")
	blocks, err := t.db.Blocks()
	if err != nil {
		return nil, err
	}

	var (
		maxTime    time.Time
		minTime    time.Time
		numSamples uint64
		numSeries  uint64
		size       int64
	)
	for _, block := range blocks {
		b, ok := block.(*tsdb.Block)
		if !ok {
			continue
		}

		var (
			blMinTime = time.Unix(0, nanoseconds(b.MinTime())).UTC()
			blMaxTime = time.Unix(0, nanoseconds(b.MaxTime())).UTC()
			truncDir  = b.Dir()[len(t.dataDir)+1:]
			blockDir  = truncDir
		)
		if i := strings.Index(truncDir, "/"); i != -1 {
			blockDir = truncDir[:strings.Index(truncDir, "/")]
		}

		_ = level.Debug(t.logger).Log("message", "checking block",
			"path", blockDir,
			"minTime", blMinTime,
			"maxTime", blMaxTime,
		)

		size += b.Size()
		numSamples += b.Meta().Stats.NumSamples
		numSeries += b.Meta().Stats.NumSeries

		if blMinTime.Before(minTime) || minTime.IsZero() {
			minTime = blMinTime
		}

		if blMaxTime.After(maxTime) || maxTime.IsZero() {
			maxTime = blMaxTime
		}
	}

	return &BlockMeta{
		Meta: &Meta{
			MaxTime:    maxTime,
			MinTime:    minTime,
			NumSamples: numSamples,
			NumSeries:  numSeries,
			Size:       size,
		},
		BlockCount: len(blocks),
	}, nil
}

func nanoseconds(milliseconds int64) int64 {
	return milliseconds * 1000000
}
