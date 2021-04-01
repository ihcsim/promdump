package k8s

import (
	"github.com/ihcsim/promdump/pkg/config"
	"github.com/ihcsim/promdump/pkg/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Clientset knows how to interact with a K8s cluster. It has a reference to
// user configuration stored in viper.
type Clientset struct {
	config    *config.Config
	k8sConfig *rest.Config
	logger    *log.Logger
	kubernetes.Interface
}

// NewClientset returns a new Clientset for the given config.
func NewClientset(appConfig *config.Config, k8sConfig *rest.Config, logger *log.Logger) (*Clientset, error) {
	k8sClientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}

	return &Clientset{
		appConfig,
		k8sConfig,
		logger,
		k8sClientset}, nil
}
