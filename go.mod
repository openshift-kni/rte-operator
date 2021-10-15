module github.com/fromanirh/rte-operator

go 1.16

require (
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/drone/envsubst v1.0.3
	github.com/go-logr/logr v0.4.0
	github.com/k8stopologyawareschedwg/deployer v0.0.8
	github.com/k8stopologyawareschedwg/resource-topology-exporter v0.2.3
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/openshift-kni/resource-topology-exporter v0.2.1
	github.com/pkg/errors v0.9.1
	k8s.io/api v0.22.2
	k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/kube-scheduler v0.21.0
	sigs.k8s.io/controller-runtime v0.9.2
	sigs.k8s.io/scheduler-plugins v0.19.9
)

replace (
	k8s.io/api => k8s.io/api v0.22.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.2
	k8s.io/apiserver => k8s.io/apiserver v0.22.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.2
	k8s.io/client-go => k8s.io/client-go v0.22.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.2
	k8s.io/code-generator => k8s.io/code-generator v0.22.2
	k8s.io/component-base => k8s.io/component-base v0.22.2
	k8s.io/component-helpers => k8s.io/component-helpers v0.22.2
	k8s.io/controller-manager => k8s.io/controller-manager v0.22.2
	k8s.io/cri-api => k8s.io/cri-api v0.22.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.2
	k8s.io/kubectl => k8s.io/kubectl v0.22.2
	k8s.io/kubelet => k8s.io/kubelet v0.22.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.2
	k8s.io/metrics => k8s.io/metrics v0.22.2
	k8s.io/mount-utils => k8s.io/mount-utils v0.22.2
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.22.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.2
)

// temporary hack
replace github.com/k8stopologyawareschedwg/deployer => github.com/fromanirh/deployer v0.0.106
