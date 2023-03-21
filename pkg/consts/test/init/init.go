package init

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	"github.com/openshift/ocm-agent-operator/pkg/test"
)

var (
	Context                = context.TODO()
	Logger                 = test.NewTestLogger().Logger()
	Scheme                 = setScheme(runtime.NewScheme())
	OCMAgentNamespacedName = types.NamespacedName{
		Name:      "ocm-agent",
		Namespace: "test-namespace",
	}
	OCMAgentHSNamespacedName = types.NamespacedName{
		Name:      "ocm-agent-hypershift",
		Namespace: "test-namespace",
	}
	TestOCMAgent = ocmagentv1alpha1.OcmAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ocm-agent",
		},
		Spec: ocmagentv1alpha1.OcmAgentSpec{
			AgentConfig: ocmagentv1alpha1.AgentConfig{
				OcmBaseUrl: "http://api.example.com",
				Services:   []string{},
			},
			OcmAgentImage: "quay.io/ocm-agent:example",
			TokenSecret:   "example-secret",
			Replicas:      1,
		},
		Status: ocmagentv1alpha1.OcmAgentStatus{},
	}
	TestHSOCMAgent = ocmagentv1alpha1.OcmAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ocm-agent-hypershift",
		},
		Spec: ocmagentv1alpha1.OcmAgentSpec{
			AgentConfig: ocmagentv1alpha1.AgentConfig{
				OcmBaseUrl: "http://api.stage.example.com",
				Services:   []string{},
			},
			OcmAgentImage: "quay.io/ocm-agent:example",
			TokenSecret:   "example-secret",
			Replicas:      1,
			FleetMode:     true,
		},
		Status: ocmagentv1alpha1.OcmAgentStatus{},
	}
	TestConfigMapSuffix = "-cm"
)

func setScheme(scheme *runtime.Scheme) *runtime.Scheme {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ocmagentv1alpha1.SchemeBuilder.AddToScheme(scheme))
	return scheme
}
