package fake

import (
	"io/ioutil"

	cristreaming "k8s.io/kubernetes/pkg/kubelet/cri/streaming"

	"github.com/ihcsim/promdump/pkg/log"
	"github.com/ihcsim/promdump/pkg/runtime"
	"github.com/ihcsim/promdump/pkg/streaming"
)

// NewStreamingServer returns a new fake streaming server instance.
func NewStreamingServer(addr string) (*streaming.Server, error) {
	var (
		runtime = runtime.NewContainerd()
		logger  = log.New(ioutil.Discard)
	)

	return streaming.New(addr, cristreaming.DefaultConfig, runtime, logger)
}
