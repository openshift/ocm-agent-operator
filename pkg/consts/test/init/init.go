package init

import (
	"context"
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
)

func setScheme(scheme *runtime.Scheme) *runtime.Scheme {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ocmagentv1alpha1.SchemeBuilder.AddToScheme(scheme))
	return scheme
}
