package server

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/ihcsim/promdump/pkg/runtime"
	"k8s.io/kubernetes/pkg/kubelet/cri/streaming"
)

const testTarget = "http://example.com"

var (
	testLogger          = log.NewNopLogger()
	testStreamingConfig = streaming.DefaultConfig

	attachURLPattern      = fmt.Sprintf(`{"url":"%s/attach/[a-zA-Z0-9_\-]*"}`, testTarget)
	execURLPattern        = fmt.Sprintf(`{"url":"%s/exec/[a-zA-Z0-9_\-]*"}`, testTarget)
	portForwardURLPattern = fmt.Sprintf(`{"url":"%s/portforward/[a-zA-Z0-9_\-]*"}`, testTarget)
)

func TestServeAttach(t *testing.T) {
	testStreamingConfig.BaseURL = &url.URL{Scheme: "http", Host: "example.com"}
	rx := regexp.MustCompile(attachURLPattern)

	var testCases = []struct {
		input        []byte
		method       string
		expectedCode int
	}{
		{
			input:        []byte(`{"container_id":"12345","stdin":true,"stdout":true,"stderr":true}`),
			method:       "POST",
			expectedCode: 200,
		},
		{
			input:        []byte(`{"container_id":"12345","stdin":true,"stdout":true,"stderr":true}`),
			method:       "GET",
			expectedCode: 200,
		},
	}

	s, err := New("", testStreamingConfig, runtime.New(), testLogger)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	for _, tc := range testCases {
		body := bytes.NewBuffer(tc.input)
		req := httptest.NewRequest(tc.method, testTarget, body)
		resp := httptest.NewRecorder()

		s.serveAttach(resp, req)
		resp.Flush()

		if actual := resp.Code; tc.expectedCode != actual {
			t.Errorf("status code mismatch. expected: %d, actual: %d", tc.expectedCode, actual)
		}

		if actual := resp.Body.String(); !rx.MatchString(actual) {
			t.Errorf("expect an alphanumeric token to be appended to result URL.\nexpected pattern: %s\nactual: %s", attachURLPattern, actual)
		}
	}
}

func TestServeExec(t *testing.T) {
	testStreamingConfig.BaseURL = &url.URL{Scheme: "http", Host: "example.com"}
	rx := regexp.MustCompile(execURLPattern)

	var testCases = []struct {
		input        []byte
		method       string
		expectedCode int
	}{
		{
			input:        []byte(`{"container_id":"12345","stdin":true,"stdout":true,"stderr":true}`),
			method:       "POST",
			expectedCode: 200,
		},
		{
			input:        []byte(`{"container_id":"12345","stdin":true,"stdout":true,"stderr":true}`),
			method:       "GET",
			expectedCode: 200,
		},
	}

	s, err := New("", testStreamingConfig, runtime.New(), testLogger)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	for _, tc := range testCases {
		body := bytes.NewBuffer(tc.input)
		req := httptest.NewRequest(tc.method, testTarget, body)
		resp := httptest.NewRecorder()

		s.serveExec(resp, req)
		resp.Flush()

		if actual := resp.Code; tc.expectedCode != actual {
			t.Errorf("status code mismatch. expected: %d, actual: %d", tc.expectedCode, actual)
		}

		if actual := resp.Body.String(); !rx.MatchString(actual) {
			t.Errorf("expect an alphanumeric token to be appended to result URL.\nexpected pattern: %s\nactual: %s", execURLPattern, actual)
		}
	}
}

func TestPortForward(t *testing.T) {
	testStreamingConfig.BaseURL = &url.URL{Scheme: "http", Host: "example.com"}
	rx := regexp.MustCompile(portForwardURLPattern)

	var testCases = []struct {
		input        []byte
		method       string
		expectedCode int
	}{
		{
			input:        []byte(`{"pod_sandbox_id":"12345","port":[9090]}`),
			method:       "POST",
			expectedCode: 200,
		},
		{
			input:        []byte(`{"pod_sandbox_id":"12345","port":[9090]}`),
			method:       "GET",
			expectedCode: 200,
		},
	}

	s, err := New("", testStreamingConfig, runtime.New(), testLogger)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	for _, tc := range testCases {
		body := bytes.NewBuffer(tc.input)
		req := httptest.NewRequest(tc.method, testTarget, body)
		resp := httptest.NewRecorder()

		s.servePortForward(resp, req)
		resp.Flush()

		if actual := resp.Code; tc.expectedCode != actual {
			t.Errorf("status code mismatch. expected: %d, actual: %d", tc.expectedCode, actual)
		}

		if actual := resp.Body.String(); !rx.MatchString(actual) {
			t.Errorf("expect an alphanumeric token to be appended to result URL.\nexpected pattern: %s\nactual: %s", portForwardURLPattern, actual)
		}
	}
}
