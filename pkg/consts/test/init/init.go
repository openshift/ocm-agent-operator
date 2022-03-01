package init

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
)

var (
	Context                = context.TODO()
	Logger                 = logr.DiscardLogger{}
	Scheme                 = setScheme(runtime.NewScheme())
	OCMAgentNamespacedName = types.NamespacedName{
		Name:      "ocm-agent",
		Namespace: "test-namespace",
	}
	TestOCMAgent = ocmagentv1alpha1.OcmAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ocm-agent",
		},
		Spec: ocmagentv1alpha1.OcmAgentSpec{
			OcmBaseUrl:     "http://api.example.com",
			Services:       []string{},
			OcmAgentImage:  "quay.io/ocm-agent:example",
			TokenSecret:    "example-secret",
			Replicas:       1,
			OcmAgentConfig: "example-config",
		},
		Status: ocmagentv1alpha1.OcmAgentStatus{},
	}
)

func setScheme(scheme *runtime.Scheme) *runtime.Scheme {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ocmagentv1alpha1.SchemeBuilder.AddToScheme(scheme))
	return scheme
}
