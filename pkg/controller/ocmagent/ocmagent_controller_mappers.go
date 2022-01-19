package ocmagent

import (
	"context"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
	oahconst "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
)

func pullSecretToOCMAgent(c client.Client, ctx context.Context, log logr.Logger) handler.MapFunc {
	return func (o client.Object) []reconcile.Request {
		var requests []reconcile.Request
		ocmagentlist := &ocmagentv1alpha1.OcmAgentList{}
		namespacedName := oahconst.BuildNamespacedName()
		if err := c.List(ctx, ocmagentlist, client.InNamespace(namespacedName.Namespace)); err != nil {
			log.Error(err,"failed to list ocmagents for pull secret")
			return requests
		}
		for _, oa := range ocmagentlist.Items {
			log.Info("queueing OcmAgent", "name", oa.Name)
			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: oa.Namespace,
					Name:      oa.Name,
				},
			}
			requests = append(requests, request)
		}
		return requests
	}
}
