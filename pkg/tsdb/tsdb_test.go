package tsdb

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ihcsim/promdump/pkg/log"
	"github.com/prometheus/prometheus/pkg/labels"
	promtsdb "github.com/prometheus/prometheus/tsdb"
)

var series []*promtsdb.MetricSample

func TestBlocks(t *testing.T) {
	logger := log.New("debug", io.Discard)
	tempDir, err := os.MkdirTemp("", "promdump-tsdb-test")
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}
	defer os.RemoveAll(tempDir)

	// initialize head and persistent blocks in data directory
	if err := initHeadBlock(tempDir); err != nil {
		t.Fatal("unexpected error when creating head block: ", err)
	}
	blockOne, blockTwo, err := initPersistentBlocks(tempDir, logger, t)
	if err != nil {
		t.Fatal("unexpected error when creating persistent blocks: ", err)
	}

	// initialize tsdb
	tsdb, err := New(tempDir, logger)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	t.Run("meta", func(t *testing.T) {
		headMeta, blockMeta, err := tsdb.Meta()
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		t.Run("head block", func(t *testing.T) {
			expectedNumSeries := 18171 // value is read from static test file
			if actual := headMeta.NumSeries; actual != uint64(expectedNumSeries) {
				t.Errorf("mismatch total series. expected: %d, actual: %d", expectedNumSeries, actual)
			}

			expectedMaxTime, err := time.Parse(timeFormat, "2021-04-18 18:28:21.05 UTC")
			if err != nil {
				t.Fatal("unexpected error: ", err)
			}
			if actual := headMeta.MaxTime; actual != expectedMaxTime {
				t.Errorf("mismatch max time. expected: %s, actual: %s", expectedMaxTime, actual)
			}

			expectedMinTime, err := time.Parse(timeFormat, "2021-04-18 13:00:05.939 UTC")
			if err != nil {
				t.Fatal("unexpected error: ", err)
			}
			if actual := headMeta.MinTime; actual != expectedMinTime {
				t.Errorf("mismatch min time. expected: %s, actual: %s", expectedMinTime, actual)
			}
		})

		t.Run("persistent blocks", func(t *testing.T) {
			expectedNumSamples := len(series)
			if actual := blockMeta.NumSamples; actual != uint64(expectedNumSamples) {
				t.Errorf("mismatch total samples. expected: %d, actual: %d", expectedNumSamples, actual)
			}

			expectedNumSeries := len(series)
			if actual := blockMeta.NumSeries; actual != uint64(expectedNumSeries) {
				t.Errorf("mismatch total series. expected: %d, actual: %d", expectedNumSeries, actual)
			}

			expectedMinTime, err := time.Parse(timeFormat, "2021-03-30 01:05:00 UTC")
			if err != nil {
				t.Fatal("unexpected error: ", err)
			}
			if actual := blockMeta.MinTime; !actual.Equal(expectedMinTime) {
				t.Errorf("mismatch min time. expected: %s, actual: %s", expectedMinTime, actual)
			}

			expectedMaxTime, err := time.Parse(timeFormat, "2021-04-01 20:52:31 UTC")
			if err != nil {
				t.Fatal("unexpected error: ", err)
			}
			if actual := blockMeta.MaxTime; !actual.Equal(expectedMaxTime) {
				t.Errorf("mismatch max time. expected: %s, actual: %s", expectedMaxTime, actual)
			}
		})
	})

	t.Run("blocks", func(t *testing.T) {
		var testCases = []struct {
			minTimeNano int64
			maxTimeNano int64
			expected    []string
		}{
			{
				minTimeNano: unix("2021-04-01 18:52:31 UTC", time.Nanosecond, t),
				maxTimeNano: unix("2021-04-01 20:52:31 UTC", time.Nanosecond, t),
				expected:    []string{blockOne},
			},
			{
				minTimeNano: unix("2021-04-01 19:00:00 UTC", time.Nanosecond, t),
				maxTimeNano: unix("2021-04-01 20:00:00 UTC", time.Nanosecond, t),
				expected:    []string{blockOne},
			},
			{
				minTimeNano: unix("2021-03-30 01:30:00 UTC", time.Nanosecond, t),
				maxTimeNano: unix("2021-03-30 02:00:00 UTC", time.Nanosecond, t),
				expected:    []string{blockTwo},
			},
			{
				minTimeNano: unix("2021-03-30 01:05:00 UTC", time.Nanosecond, t),
				maxTimeNano: unix("2021-03-30 03:05:00 UTC", time.Nanosecond, t),
				expected:    []string{blockTwo},
			},
			{
				minTimeNano: unix("2021-01-01 00:00:00 UTC", time.Nanosecond, t),
				maxTimeNano: unix("2021-01-01 02:16:00 UTC", time.Nanosecond, t),
				expected:    []string{},
			},
			{
				minTimeNano: unix("2021-04-01 16:00:00 UTC", time.Nanosecond, t),
				maxTimeNano: unix("2021-04-01 19:12:10 UTC", time.Nanosecond, t),
				expected:    []string{blockOne},
			},
			{
				minTimeNano: unix("2021-04-01 19:00:00 UTC", time.Nanosecond, t),
				maxTimeNano: unix("2021-04-01 22:48:09 UTC", time.Nanosecond, t),
				expected:    []string{blockOne},
			},
			{
				minTimeNano: unix("2021-03-30 00:00:00 UTC", time.Nanosecond, t),
				maxTimeNano: unix("2021-03-30 02:58:09 UTC", time.Nanosecond, t),
				expected:    []string{blockTwo},
			},
			{
				minTimeNano: unix("2021-03-30 02:00:00 UTC", time.Nanosecond, t),
				maxTimeNano: unix("2021-03-30 04:18:09 UTC", time.Nanosecond, t),
				expected:    []string{blockTwo},
			},
			{
				minTimeNano: unix("2021-03-30 01:05:00 UTC", time.Nanosecond, t),
				maxTimeNano: unix("2021-04-01 20:52:31 UTC", time.Nanosecond, t),
				expected:    []string{blockOne, blockTwo},
			},
			{
				minTimeNano: unix("2021-03-30 00:00:00 UTC", time.Nanosecond, t),
				maxTimeNano: unix("2021-04-02 00:00:00 UTC", time.Nanosecond, t),
				expected:    []string{blockOne, blockTwo},
			},
		}

		for i, tc := range testCases {
			t.Run(fmt.Sprintf("test case #%d", i), func(t *testing.T) {
				actual, err := tsdb.Blocks(tc.minTimeNano, tc.maxTimeNano)
				if err != nil {
					t.Fatal("unexpected error: ", err)
				}

				if len(tc.expected) == 0 && len(actual) == 0 {
					return
				}

				var found bool
			LOOP:
				for _, path := range tc.expected {
					for _, block := range actual {
						if path == block.Dir() {
							found = true
							break LOOP
						}
					}
				}
				if !found {
					t.Errorf("missing expected blocks: %v\n actual: %v", tc.expected, actual)
				}
			})
		}
	})
}

func initHeadBlock(tempDir string) error {
	// copy checkpoint test data to tempDir
	walDir := filepath.Join(tempDir, "wal")
	checkpointDir := filepath.Join(walDir, "checkpoint.00000036")
	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		return err
	}

	checkpointTestFile := filepath.Join("testdata", "wal", "checkpoint.00000036", "00000000")
	checkpointTestdata, err := os.ReadFile(checkpointTestFile)
	if err != nil {
		return err
	}

	checkpointFile := filepath.Join(checkpointDir, "00000037")
	if err := os.WriteFile(checkpointFile, checkpointTestdata, 0600); err != nil {
		return err
	}

	// copy wal test data to tempDir
	for _, testFile := range []string{"00000037", "00000038", "00000039"} {
		wal, err := os.ReadFile(filepath.Join("testdata", "wal", testFile))
		if err != nil {
			return err
		}

		walFile := filepath.Join(walDir, testFile)
		if err := os.WriteFile(walFile, wal, 0600); err != nil {
			return err
		}
	}

	return nil
}

func initPersistentBlocks(tempDir string, logger *log.Logger, t *testing.T) (string, string, error) {
	series = []*promtsdb.MetricSample{
		{
			TimestampMs: unix("2021-04-01 20:30:31 UTC", time.Millisecond, t),
			Labels: []labels.Label{
				{Name: "job", Value: "tsdb"},
				{Name: "app", Value: "app-00"},
			},
		},
		{
			TimestampMs: unix("2021-04-01 20:00:31 UTC", time.Millisecond, t),
			Labels: []labels.Label{
				{Name: "job", Value: "tsdb"},
				{Name: "app", Value: "app-01"},
			},
		},
		{
			TimestampMs: unix("2021-04-01 19:33:00 UTC", time.Millisecond, t),
			Labels: []labels.Label{
				{Name: "job", Value: "tsdb"},
				{Name: "app", Value: "app-02"},
			},
		},
		{
			TimestampMs: unix("2021-03-30 01:06:47 UTC", time.Millisecond, t),
			Labels: []labels.Label{
				{Name: "job", Value: "tsdb"},
				{Name: "app", Value: "app-00"},
			},
		},
		{
			TimestampMs: unix("2021-03-30 02:36:00 UTC", time.Millisecond, t),
			Labels: []labels.Label{
				{Name: "job", Value: "tsdb"},
				{Name: "app", Value: "app-01"},
			},
		},
		{
			TimestampMs: unix("2021-03-30 03:01:48 UTC", time.Millisecond, t),
			Labels: []labels.Label{
				{Name: "job", Value: "tsdb"},
				{Name: "app", Value: "app-02"},
			},
		},
	}

	blockOne, err := promtsdb.CreateBlock(series[:3], tempDir,
		unix("2021-04-01 18:52:31 UTC", time.Millisecond, t),
		unix("2021-04-01 20:52:31 UTC", time.Millisecond, t),
		logger.Logger)
	if err != nil {
		return "", "", err
	}

	blockTwo, err := promtsdb.CreateBlock(series[3:], tempDir,
		unix("2021-03-30 01:05:00 UTC", time.Millisecond, t),
		unix("2021-03-30 03:05:00 UTC", time.Millisecond, t),
		logger.Logger)
	if err != nil {
		return "", "", err
	}

	return blockOne, blockTwo, err
}

const timeFormat = "2006-01-02 15:04:05 MST"

func unix(date string, d time.Duration, t *testing.T) int64 {
	parsed, err := time.Parse(timeFormat, date)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	switch d {
	case time.Millisecond:
		return parsed.Unix() * 1000
	case time.Nanosecond:
		return parsed.UnixNano()
	default:
		t.Fatal("unsupported duration type: ", d)
	}

	return 0
}
