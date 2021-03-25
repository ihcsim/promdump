module github.com/ihcsim/promdump

go 1.14

require (
	github.com/Azure/azure-sdk-for-go v51.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.0.1 // indirect
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/alecthomas/units v0.0.0-20210208195552-ff826a37aa15 // indirect
	github.com/aws/aws-sdk-go v1.37.8 // indirect
	github.com/cenkalti/backoff/v4 v4.0.2 // indirect
	github.com/containerd/containerd v1.4.3 // indirect
	github.com/dgryski/go-sip13 v0.0.0-20200911182023-62edffca9245 // indirect
	github.com/digitalocean/godo v1.57.0 // indirect
	github.com/docker/docker v20.10.3+incompatible
	github.com/go-kit/kit v0.10.0
	github.com/go-openapi/analysis v0.19.8 // indirect
	github.com/go-openapi/errors v0.19.9
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/runtime v0.19.15 // indirect
	github.com/go-openapi/strfmt v0.20.0 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/go-zookeeper/zk v1.0.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.2 // indirect
	github.com/google/pprof v0.0.0-20210208152844-1612e9be7af6 // indirect
	github.com/gophercloud/gophercloud v0.15.0 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/hashicorp/consul/api v1.8.1 // indirect
	github.com/hetznercloud/hcloud-go v1.23.1 // indirect
	github.com/influxdata/influxdb v1.8.4 // indirect
	github.com/julienschmidt/httprouter v1.3.0
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/miekg/dns v1.1.38 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/opentracing-contrib/go-stdlib v1.0.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/prometheus/alertmanager v0.20.0 // indirect
	github.com/prometheus/client_golang v1.9.0 // indirect
	github.com/prometheus/common v0.18.0 // indirect
	github.com/prometheus/exporter-toolkit v0.5.1 // indirect
	github.com/prometheus/prometheus v1.8.1
	github.com/rs/cors v1.7.0 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20200824052919-0d455de96546 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/uber/jaeger-client-go v2.25.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.0+incompatible // indirect
	github.com/xlab/treeprint v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.4.6 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/net v0.0.0-20210324205630-d1beb07c2056 // indirect
	golang.org/x/oauth2 v0.0.0-20210210192628-66670185b0cd // indirect
	golang.org/x/text v0.3.5 // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	golang.org/x/tools v0.1.0 // indirect
	google.golang.org/api v0.39.0 // indirect
	gopkg.in/fsnotify/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	gotest.tools/v3 v3.0.3 // indirect
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/cli-runtime v0.20.4
	k8s.io/client-go v0.20.4
	k8s.io/cri-api v0.20.4
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.5.0 // indirect
	k8s.io/kubernetes v1.20.4
	sigs.k8s.io/controller-runtime v0.8.2
)

// see https://github.com/golang/go/issues/32776
replace (
	k8s.io/api => k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.4
	k8s.io/apiserver => k8s.io/apiserver v0.20.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.4
	k8s.io/client-go => k8s.io/client-go v0.20.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.4
	k8s.io/code-generator => k8s.io/code-generator v0.20.4
	k8s.io/component-base => k8s.io/component-base v0.20.4
	k8s.io/component-helpers => k8s.io/component-helpers v0.20.4
	k8s.io/controller-manager => k8s.io/controller-manager v0.20.4
	k8s.io/cri-api => k8s.io/cri-api v0.20.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.20.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.20.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.20.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.20.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.20.4
	k8s.io/kubectl => k8s.io/kubectl v0.20.4
	k8s.io/kubelet => k8s.io/kubelet v0.20.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.20.4
	k8s.io/metrics => k8s.io/metrics v0.20.4
	k8s.io/mount-utils => k8s.io/mount-utils v0.20.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.20.4
)
