package tsdb

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ihcsim/promdump/pkg/log"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/tsdb"
	promtsdb "github.com/prometheus/prometheus/tsdb"
)

func TestBlocks(t *testing.T) {
	logger := log.New("debug", ioutil.Discard)
	tempDir, err := ioutil.TempDir("", "promdump-tsdb-test")
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}
	defer os.RemoveAll(tempDir)

	series := []*tsdb.MetricSample{
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

	tsdb := New(tempDir, logger)

	blockOne, err := promtsdb.CreateBlock(series[:3], tempDir,
		unix("2021-04-01 18:52:31 UTC", time.Millisecond, t),
		unix("2021-04-01 20:52:31 UTC", time.Millisecond, t),
		logger.Logger)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	blockTwo, err := promtsdb.CreateBlock(series[3:], tempDir,
		unix("2021-03-30 01:05:00 UTC", time.Millisecond, t),
		unix("2021-03-30 03:05:00 UTC", time.Millisecond, t),
		logger.Logger)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	t.Run("meta", func(t *testing.T) {
		meta, err := tsdb.Meta()
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		expectedTotalSamples := len(series)
		if actual := meta.TotalSamples; actual != uint64(expectedTotalSamples) {
			t.Errorf("mismatch total samples. expected: %d, actual: %d", expectedTotalSamples, actual)
		}

		expectedTotalSeries := len(series)
		if actual := meta.TotalSeries; actual != uint64(expectedTotalSeries) {
			t.Errorf("mismatch total series. expected: %d, actual: %d", expectedTotalSeries, actual)
		}

		expectedStartTime, err := time.Parse(timeFormat, "2021-03-30 01:05:00 UTC")
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}
		if actual := meta.Start; !actual.Equal(expectedStartTime) {
			t.Errorf("mismatch start time. expected: %s, actual: %s", expectedStartTime, actual)
		}

		expectedEndTime, err := time.Parse(timeFormat, "2021-04-01 20:52:31 UTC")
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}
		if actual := meta.End; !actual.Equal(expectedEndTime) {
			t.Errorf("mismatch end time. expected: %s, actual: %s", expectedEndTime, actual)
		}
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
