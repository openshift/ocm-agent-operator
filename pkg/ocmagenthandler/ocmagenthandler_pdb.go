package ocmagenthandler

import (
	"context"
	"reflect"

	ocmagentv1alpha1 "github.com/openshift/ocm-agent-operator/api/v1alpha1"
	oah "github.com/openshift/ocm-agent-operator/pkg/consts/ocmagenthandler"
	v1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func buildOCMAgentPodDisruptionBudget(ocmAgent ocmagentv1alpha1.OcmAgent) *v1.PodDisruptionBudget {
	namespacedName := oah.BuildNamespacedName(ocmAgent.Name + oah.PDBSuffix)

	return &v1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Spec: v1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": ocmAgent.Name,
				},
			},
		},
	}
}

// ensurePodDisruptionBudget ensures that an OCMAgent PDB exists on the cluster
// and that its configuration matches what is expected.
func (o *ocmAgentHandler) ensurePodDisruptionBudget(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	pdb := buildOCMAgentPodDisruptionBudget(ocmAgent)
	foundPDB := &v1.PodDisruptionBudget{}

	// Check if the PDB already exists
	if err := o.Client.Get(context.TODO(), types.NamespacedName{
		Name: pdb.Name, Namespace: pdb.Namespace}, foundPDB); err != nil {

		if k8serrors.IsNotFound(err) {
			o.Log.Info("A Pod Disruption Budget does not exist; will be created", "PDB.Namespace", pdb.Namespace, "PDB.Name", pdb.Name)
			// Set the controller reference
			if err := controllerutil.SetControllerReference(&ocmAgent, pdb, o.Scheme); err != nil {
				return err
			}
			// and create it now
			err = o.Client.Create(context.TODO(), pdb)

			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if !reflect.DeepEqual(foundPDB.Spec.MinAvailable, pdb.Spec.MinAvailable) {
			foundPDB.Spec = *pdb.Spec.DeepCopy()
			o.Log.Info("Updating Pod Disruption Budget", "PDB.Namespace", foundPDB.Namespace, "PDB.Name", foundPDB.Name)
			err = o.Client.Update(context.TODO(), foundPDB)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ensurePodDisruptionBudgetDeleted removes the PDB from the cluster
func (o *ocmAgentHandler) ensurePodDisruptionBudgetDeleted(ocmAgent ocmagentv1alpha1.OcmAgent) error {
	pdb := buildOCMAgentPodDisruptionBudget(ocmAgent)
	foundPDB := &v1.PodDisruptionBudget{}

	if err := o.Client.Get(context.TODO(), types.NamespacedName{
		Name: pdb.Name, Namespace: pdb.Namespace}, foundPDB); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	o.Log.Info("Ensuring Pod Disruption Budget is removed", "PDB.Namespace", foundPDB.Namespace, "PDB.Name", foundPDB.Name)
	return o.Client.Delete(context.TODO(), foundPDB)
}
