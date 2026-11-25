module github.com/lim123123123/concolic-testing

go 1.27.1

require (
	github.com/ezrantn/go-z3 v0.1.0-beta
	github.com/samber/lo v1.53.0
	github.com/stretchr/testify v1.12.1
	go.chromium.org/luci v0.0.0-20261118063837-0cfda84e9220
	go.uber.org/zap v1.28.0
	golang.org/x/tools v0.52.0
	gonum.org/v1/gonum v0.17.0
	k8s.io/apimachinery v0.37.2
	sigs.k8s.io/controller-runtime v0.6.5
	sigs.k8s.io/yaml v1.6.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/evanphx/json-patch v4.9.0+incompatible // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.58.0 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/term v0.47.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.37.0 // indirect
	k8s.io/client-go v0.37.0 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/kube-openapi v0.0.0-20260721132016-d427ff9ee9ad // indirect
	k8s.io/utils v0.0.0-20260626114624-be93311217bd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
)

replace (
	github.com/ezrantn/go-z3 => ../go-z3

	// Same versions as zookeeper-operator, controller-runtime v0.6.5 requires k8s libraries of the 0.19 era
	k8s.io/api => k8s.io/api v0.19.13
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.13
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.14-rc.0
	k8s.io/client-go => k8s.io/client-go v0.19.13
)
