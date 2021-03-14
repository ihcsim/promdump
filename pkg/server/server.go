package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/cri/streaming"
)

const (
	defaultExitTimeout   = time.Minute * 2
	formValueContainerID = "containerID"
)

// Server exposes a number of HTTP endpoints to support container exec, attach
// and port-forward functionalities.
type Server struct {
	logger      log.Logger
	exitTimeout time.Duration
	http.Server
	stream streaming.Server
}

// New returns a new instance of Server.
func New(addr string, config streaming.Config, runtime streaming.Runtime, logger log.Logger) (*Server, error) {
	stream, err := streaming.NewServer(config, runtime)
	if err != nil {
		return nil, err
	}

	s := &Server{
		exitTimeout: defaultExitTimeout,
		logger:      log.With(logger, "component", "server"),
		stream:      stream,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/attach", s.serveAttach)
	mux.HandleFunc("/exec", s.serveExec)
	mux.HandleFunc("/portforward", s.servePortForward)
	mux.Handle("/attach/", stream)
	mux.Handle("/exec/", stream)
	mux.Handle("/portforward/", stream)
	s.Server = http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s, nil
}

// Start
func (s *Server) Start() error {
	s.logger.Log("status", "starting server", "listenAt", s.Addr)
	return s.ListenAndServe()
}

// Stop
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(
		context.Background(), s.exitTimeout)
	defer func() {
		cancel()
	}()

	s.logger.Log("status", "shutting down")
	return s.Shutdown(ctx)
}

func (s *Server) serveAttach(resp http.ResponseWriter, req *http.Request) {
	attachResponse := func(requestBody []byte) ([]byte, error, int) {
		var attachReq runtimeapi.AttachRequest
		if err := json.Unmarshal(requestBody, &attachReq); err != nil {
			return nil, err, http.StatusBadRequest
		}

		attachResp, err := s.stream.GetAttach(&attachReq)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}

		data, err := json.Marshal(attachResp)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}

		return data, nil, http.StatusOK
	}

	s.streamingURL(resp, req, attachResponse)
}

func (s *Server) serveExec(resp http.ResponseWriter, req *http.Request) {
	execResponse := func(requestData []byte) ([]byte, error, int) {
		var execReq runtimeapi.ExecRequest
		if err := json.Unmarshal(requestData, &execReq); err != nil {
			return nil, err, http.StatusBadRequest
		}

		execResp, err := s.stream.GetExec(&execReq)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}

		data, err := json.Marshal(execResp)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}

		return data, nil, http.StatusOK
	}

	s.streamingURL(resp, req, execResponse)
}

func (s *Server) servePortForward(resp http.ResponseWriter, req *http.Request) {
	portForwardResponse := func(requestData []byte) ([]byte, error, int) {
		var portForwardReq runtimeapi.PortForwardRequest
		if err := json.Unmarshal(requestData, &portForwardReq); err != nil {
			return nil, err, http.StatusBadRequest
		}

		portForwardResp, err := s.stream.GetPortForward(&portForwardReq)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}

		data, err := json.Marshal(portForwardResp)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}

		return data, nil, http.StatusOK
	}

	s.streamingURL(resp, req, portForwardResponse)
}

func (s *Server) streamingURL(resp http.ResponseWriter, req *http.Request, responseData func(input []byte) ([]byte, error, int)) {
	body := make([]byte, req.ContentLength)

	var (
		responseCode int
		responseBody string
	)
	defer func() {
		s.logger.Log("responseCode", responseCode, "responseBody", responseBody)
	}()

	if _, err := req.Body.Read(body); err != nil {
		responseCode, responseBody = http.StatusBadRequest, err.Error()
		http.Error(resp, responseBody, responseCode)
		return
	}
	defer req.Body.Close()
	s.logger.Log("requestBody", string(body))

	data, err, statusCode := responseData(body)
	if err != nil {
		responseCode, responseBody = statusCode, err.Error()
		http.Error(resp, responseBody, responseCode)
		return
	}

	if _, err := resp.Write(data); err != nil {
		responseCode, responseBody = http.StatusInternalServerError, err.Error()
		http.Error(resp, responseBody, responseCode)
		return
	}

	responseCode, responseBody = http.StatusOK, string(data)
}

func handleErrs(errChan <-chan error) error {
	if len(errChan) > 0 {
		var errs error
		for err := range errChan {
			errs = fmt.Errorf("%s\n%s", errs, err)
		}
		return errs
	}

	return nil
}
