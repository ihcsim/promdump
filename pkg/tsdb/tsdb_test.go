package tsdb

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ihcsim/promdump/pkg/log"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/tsdb"
	promtsdb "github.com/prometheus/prometheus/tsdb"
)

func TestBlocks(t *testing.T) {
	logger := log.New(ioutil.Discard)
	tempDir, err := ioutil.TempDir("", "promdump-tdsb-test")
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}
	defer os.RemoveAll(tempDir)

	var (
		series = map[int][]*tsdb.MetricSample{
			0: []*tsdb.MetricSample{
				// 2021-04-01 20:30:31 GMT
				{
					TimestampMs: 1617309031000,
					Labels:      []labels.Label{{Name: "job", Value: "tsdb"}},
				},

				// 2021-04-01 20:00:31 GMT
				{
					TimestampMs: 1617307231000,
					Labels:      []labels.Label{{Name: "job", Value: "tsdb"}},
				},

				// 2021-04-01 19:31:00 GMT
				{
					TimestampMs: 1617305631000,
					Labels:      []labels.Label{{Name: "job", Value: "tsdb"}},
				},
			},

			1: []*tsdb.MetricSample{
				// 2021-03-30 01:06:47 GMT
				{
					TimestampMs: 1617066407000,
					Labels:      []labels.Label{{Name: "job", Value: "tsdb"}},
				},

				// 2021-03-30 02:36:00 GMT
				{
					TimestampMs: 1617071760000,
					Labels:      []labels.Label{{Name: "job", Value: "tsdb"}},
				},

				// 2021-03-30 03:01:46 GMT
				{
					TimestampMs: 1617073308000,
					Labels:      []labels.Label{{Name: "job", Value: "tsdb"}},
				},
			},
		}
	)

	var timeRanges = []struct {
		minTimeMs int64
		maxTimeMs int64
	}{
		// 2021-04-01 20:52:31 GMT to 2021-04-01 18:52:31 GMT
		{
			minTimeMs: 1617303151000,
			maxTimeMs: 1617310351000,
		},

		// 2021-03-30 01:05:00 GMT to 2021-03-30 03:05:00 GMT
		{
			minTimeMs: 1617066300000,
			maxTimeMs: 1617073500000,
		},
	}

	tsdb := New(tempDir, logger)
	for i, tr := range timeRanges {
		if _, err := promtsdb.CreateBlock(series[i], tempDir, tr.minTimeMs, tr.maxTimeMs, logger.Logger); err != nil {
			t.Fatalf("test case %d has unexpected error: %s", i, err)
		}

		blocks, err := tsdb.Blocks(nanoseconds(tr.minTimeMs), nanoseconds(tr.maxTimeMs))
		if err != nil {
			t.Fatalf("test case %d has unexpected error: %s", i, err)
		}

		expected := 1
		if actual := len(blocks); expected != actual {
			t.Errorf("number of blocks mismatch. expected: %d, actual: %d", expected, actual)
		}
	}
}
