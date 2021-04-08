package download

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/ihcsim/promdump/pkg/log"
)

var errChecksumMismatch = fmt.Errorf("mismatch checksum")

// Download can issue GET requests to download assets from a remote endpoint.
type Download struct {
	logger   *log.Logger
	http     *http.Client
	localDir string
}

// New returns a new instance of Download.
func New(localDir string, timeout time.Duration, logger *log.Logger) *Download {
	return &Download{
		localDir: localDir,
		logger:   logger,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

// Get issues a GET request to the remote endpoint to download the promdump TAR
// file, to the localDir directory. If non-empty, it also fetches the SHA256 sum
// file from the specifed endpoint, and used that to verified the content of the
// downloaded TAR file.
// If the file is already present on the local file system, then the download will
// be skipped, unless force is set to true to trigger a re-download.
// The content of the file is then read and returned. Caller is responsible for
// closing the returned ReadCloser.
func (d *Download) Get(force bool, remoteURI, remoteURISHA string) (io.ReadCloser, error) {
	var (
		exists    = true
		filename  = path.Base(remoteURI)
		savedPath = filepath.Join(d.localDir, filename)
	)

	tarFile, err := os.Open(savedPath)
	if err != nil {
		// file exists; errors caused by something else
		if !os.IsNotExist(err) {
			return nil, err
		}

		exists = false
	}

	if exists && !force {
		return tarFile, nil
	}

	if err := d.download(remoteURI, savedPath); err != nil {
		return nil, err
	}

	newFile, err := os.Open(savedPath)
	if err != nil {
		return nil, err
	}

	if err := d.checksum(newFile, remoteURISHA); err != nil {
		return nil, err
	}

	return newFile, nil
}

func (d *Download) download(remote, savedPath string) error {
	_ = level.Info(d.logger).Log("message", "downloading promdump",
		"remoteURI", remote,
		"timeout", d.http.Timeout,
		"localDir", savedPath)

	resp, err := d.http.Get(remote)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed. reason: %s", resp.Status)
	}

	file, err := os.Create(savedPath)
	if err != nil {
		return err
	}

	nbr, err := io.CopyN(file, resp.Body, resp.ContentLength)
	if err != nil {
		return err
	}

	_ = level.Info(d.logger).Log("message", "download completed", "numBytesWrite", nbr)
	return nil
}

func (d *Download) checksum(file *os.File, remote string) error {
	_ = level.Info(d.logger).Log("message", "verifying checksum",
		"endpoint", remote,
		"timeout", d.http.Timeout,
	)

	resp, err := d.http.Get(remote)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	expected := &bytes.Buffer{}
	if _, err := io.CopyN(expected, resp.Body, resp.ContentLength); err != nil {
		return err
	}

	sha := sha256.New()
	if _, err := io.Copy(sha, file); err != nil {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if actual := fmt.Sprintf("%x", sha.Sum(nil)); expected.String() == string(actual) {
		return fmt.Errorf("%w: expected:%s, actual:%s", errChecksumMismatch, expected, actual)
	}

	_ = level.Info(d.logger).Log("message", "confirmed checksum")
	return nil
}
