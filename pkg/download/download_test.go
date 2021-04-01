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
		dirname = "promdump-test"
		force   = false
		logger  = log.New(ioutil.Discard)
		timeout = time.Second

		mux          = http.NewServeMux()
		server       = httptest.NewServer(mux)
		remoteURI    = server.URL
		remoteURISHA = fmt.Sprintf("%s/checksum", server.URL)

		response    = []byte("test response data")
		responseSHA = sha256.Sum256(response)
	)

	mux.Handle("/", http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		if _, err := resp.Write(response); err != nil {
			t.Fatal("unexpected error: ", err)
		}
	}))
	mux.Handle("/checksum", http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)

		d := make([]byte, sha256.Size)
		for i, b := range responseSHA {
			d[i] = b
		}
		fmt.Printf("%x\n", d)
		if _, err := resp.Write(d); err != nil {
			t.Fatal("unexpected error: ", err)
		}
	}))

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

	received, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	if !reflect.DeepEqual(received, response) {
		t.Errorf("mismatch response. expected:%s, actual:%s", received, response)
	}
}
