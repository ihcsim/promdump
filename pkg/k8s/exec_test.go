package k8s

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"testing"

	authzv1 "k8s.io/api/authorization/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	fakerest "k8s.io/client-go/rest/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/ihcsim/promdump/pkg/config"
	"github.com/ihcsim/promdump/pkg/k8s/fake"
	"github.com/ihcsim/promdump/pkg/log"
	"github.com/spf13/viper"
)

func TestCanExec(t *testing.T) {

	var testCases = []struct {
		allowed  bool
		reason   string
		expected error
	}{
		{allowed: true, reason: "allowed"},
		{allowed: false, reason: "denied", expected: deniedCreateExecErr},
	}

	for _, tc := range testCases {
		t.Run(tc.reason, func(t *testing.T) {
			reaction := func(action k8stesting.Action) (handled bool, ret apiruntime.Object, err error) {
				return true,
					&authzv1.SelfSubjectAccessReview{
						Status: authzv1.SubjectAccessReviewStatus{
							Allowed: tc.allowed,
							Reason:  tc.reason,
						},
					}, nil
			}

			k8sClientset := &k8sfake.Clientset{}
			k8sClientset.Fake.AddReactor("create", "selfsubjectaccessreviews", reaction)
			clientset := Clientset{
				&config.Config{Viper: viper.New()},
				&rest.Config{},
				log.New("debug", ioutil.Discard),
				k8sClientset,
			}

			if actual := clientset.CanExec(); !errors.Is(actual, tc.expected) {
				t.Errorf("mismatch errors: expected: %v, actual: %v", tc.expected, actual)
			}
		})
	}
}

func TestExec(t *testing.T) {
	testConfig := &config.Config{Viper: viper.New()}
	testConfig.Set("namespace", "test-ns")
	testConfig.Set("pod", "test-pod")
	testConfig.Set("container", "test-container")
	testConfig.Set("request-timeout", "5s")
	testConfig.Set("min-time", "2000-01-01 00:00:00")
	testConfig.Set("max-time", "2000-01-02 00:00:00")

	execRoundTripper := func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
		}, nil
	}

	restClient := &fakerest.RESTClient{
		Client: fakerest.CreateHTTPClient(execRoundTripper),
	}
	k8sClientset := kubernetes.New(restClient)
	clientset := &Clientset{
		testConfig,
		&rest.Config{},
		log.New("debug", ioutil.Discard),
		k8sClientset,
	}

	// fake the executor constructor so that the fake executor is used in the
	// ExecPod() method
	newExecutor = newFakeExecutor

	cmd := []string{}
	if err := clientset.ExecPod(cmd, os.Stdin, os.Stdout, os.Stderr, false); err != nil {
		t.Fatal("unexpected error: ", err)
	}
}

func newFakeExecutor(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
	return fake.NewExecutor(url), nil
}
