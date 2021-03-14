package main

import (
	"os"
	"os/signal"

	"github.com/ihcsim/promdump/pkg/log"
	"github.com/ihcsim/promdump/pkg/runtime"
	"github.com/ihcsim/promdump/pkg/server"
	"k8s.io/kubernetes/pkg/kubelet/cri/streaming"
)

var logger = log.New(os.Stderr)

func main() {
	var (
		addr    = ":5078"
		config  = streaming.DefaultConfig
		runtime = runtime.New()
	)

	server, err := server.New(addr, config, runtime, logger)
	if err != nil {
		exit(err)
	}

	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Kill, os.Interrupt)
	go func() {
		<-kill
		server.Stop()
	}()

	logger.Log("error", server.Start())
}

func exit(err error) {
	logger.Log("error", err)
	os.Exit(1)
}
