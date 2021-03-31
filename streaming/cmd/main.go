package main

import (
	"os"
	"os/signal"

	"github.com/ihcsim/promdump/pkg/log"
	"github.com/ihcsim/promdump/pkg/runtime"
	"github.com/ihcsim/promdump/pkg/streaming"
	k8sstream "k8s.io/kubernetes/pkg/kubelet/cri/streaming"
)

var logger = log.New(os.Stderr).With("component", "streaming")

func main() {
	var (
		addr    = ":5078"
		config  = k8sstream.DefaultConfig
		runtime = runtime.NewContainerd()
	)

	streaming, err := streaming.New(addr, config, runtime, logger)
	if err != nil {
		exit(err)
	}

	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt)
	go func() {
		<-kill
		if err := streaming.Stop(); err != nil {
			logger.Log("error", err)
		}
	}()

	logger.Log("error", streaming.Start())
}

func exit(err error) {
	logger.Log("error", err)
	os.Exit(1)
}
