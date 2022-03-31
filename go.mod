module github.com/openshift/ocm-agent-operator

go 1.16

require (
	github.com/coreos/prometheus-operator v0.38.1-0.20200424145508-7e176fda06cc
	github.com/emicklei/go-restful v2.10.0+incompatible // indirect
	github.com/go-logr/logr v0.4.0
	github.com/gobuffalo/flect v0.2.2 // indirect
	github.com/golang/mock v1.6.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/openshift/api v3.9.1-0.20190424152011-77b8897ec79a+incompatible
	github.com/openshift/operator-custom-metrics v0.4.2
	github.com/operator-framework/operator-sdk v0.18.2
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20210421082810-95288971da7e
	sigs.k8s.io/controller-runtime v0.10.0
	sigs.k8s.io/controller-tools v0.3.0
)

replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210910062324-a41d3573a3ba
	k8s.io/api => k8s.io/api v0.21.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.1
	k8s.io/client-go => k8s.io/client-go v0.21.1
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20201125052318-b85a18cbf338
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.0.0-20210209143830-3442c7a36c1e
)
