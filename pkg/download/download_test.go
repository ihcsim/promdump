package download

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/ihcsim/promdump/pkg/log"
)

func TestDownload(t *testing.T) {
	var (
		dirname         = "promdump-test"
		downloadContent = []byte("test response data")
		force           = false
		logger          = log.New("debug", ioutil.Discard)
		timeout         = time.Second

		mux          = http.NewServeMux()
		server       = httptest.NewServer(mux)
		remoteURI    = server.URL
		remoteURISHA = fmt.Sprintf("%s/checksum", server.URL)
	)

	// returns the download content
	mux.Handle("/", http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		if _, err := resp.Write(downloadContent); err != nil {
			t.Fatal("unexpected error: ", err)
		}
	}))

	// returns the checksum of the download content
	mux.Handle("/checksum", http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		h := sha256.New()
		if _, err := h.Write(downloadContent); err != nil {
			t.Fatal("unexpected error: ", err)
		}
		d := fmt.Sprintf("%x", h.Sum(nil))

		resp.WriteHeader(http.StatusOK)
		if _, err := resp.Write([]byte(d)); err != nil {
			t.Fatal("unexpected error: ", err)
		}
	}))

	// the downloaded content will be saved in tempDir
	tempDir, err := ioutil.TempDir("", dirname)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}
	defer os.RemoveAll(tempDir)

	d := New(tempDir, timeout, logger)
	reader, err := d.Get(force, remoteURI, remoteURISHA)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	// read and compare the downloaded content with the actual data
	actual, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	if !reflect.DeepEqual(actual, downloadContent) {
		t.Errorf("mismatch response. expected:%s, actual:%s", downloadContent, actual)
	}
}
