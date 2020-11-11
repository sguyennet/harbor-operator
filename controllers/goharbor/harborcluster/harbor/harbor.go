package harbor

import (
	"fmt"
	"github.com/goharbor/harbor-operator/apis/goharbor.io/v1alpha2"
	"github.com/goharbor/harbor-operator/pkg/k8s"
	"github.com/goharbor/harbor-operator/pkg/lcm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/kustomize/kstatus/status"
)

type HarborController struct {
	k8s.Client
}

func (harbor *HarborController) Apply(harborcluster *v1alpha2.HarborCluster) (*lcm.CRStatus, error) {
	var harborCR *v1alpha2.Harbor
	err := harbor.Get(harbor.getHarborCRNamespacedName(harborcluster), harborCR)
	if err != nil {
		if errors.IsNotFound(err) {
			harborCR = harbor.getHarborCR(harborcluster)
			err = harbor.Create(harborCR)
			if err != nil {
				return harborClusterCRNotReadyStatus(CreateHarborCRError, err.Error()), err
			}
		} else {
			return harborClusterCRNotReadyStatus(GetHarborCRError, err.Error()), err
		}
	} else {
		harborCR = harbor.getHarborCR(harborcluster)
		err = harbor.Update(harborCR)
		if err != nil {
			return harborClusterCRNotReadyStatus(UpdateHarborCRError, err.Error()), err
		}
	}

	return harborClusterCRStatus(harborCR), nil
}

func (harbor *HarborController) Delete(harborcluster *v1alpha2.HarborCluster) (*lcm.CRStatus, error) {
	panic("implement me")
}

func (harbor *HarborController) Upgrade(harborcluster *v1alpha2.HarborCluster) (*lcm.CRStatus, error) {
	panic("implement me")
}

func NewHarborController() lcm.Controller {
	return &HarborController{}
}

// getHarborCR will get a Harbor CR from the harborcluster definition
func (harbor *HarborController) getHarborCR(harborcluster *v1alpha2.HarborCluster) *v1alpha2.Harbor {
	namespacedName := harbor.getHarborCRNamespacedName(harborcluster)

	harborCR := &v1alpha2.Harbor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
			Labels: map[string]string{
				k8s.HarborClusterNameLabel: harborcluster.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(harborcluster, v1alpha2.HarborClusterGVK),
			},
		},
		Spec: *harborcluster.Spec.HarborSpec.DeepCopy(),
	}
	// todo to filled with database, cache and storage spec.
	return harborCR
}

func harborClusterCRNotReadyStatus(reason, message string) *lcm.CRStatus {
	return lcm.New(v1alpha2.ServiceReady).WithStatus(corev1.ConditionFalse).WithReason(reason).WithMessage(message)
}

func harborClusterCRUnknownStatus(reason, message string) *lcm.CRStatus {
	return lcm.New(v1alpha2.ServiceReady).WithStatus(corev1.ConditionUnknown).WithReason(reason).WithMessage(message)
}

// harborClusterCRStatus will assembly the harbor cluster status according the v1alpha1.Harbor status
func harborClusterCRStatus(harbor *v1alpha2.Harbor) *lcm.CRStatus {
	for _, condition := range harbor.Status.Conditions {
		if condition.Type == status.ConditionInProgress {
			return lcm.New(v1alpha2.ServiceReady).WithStatus(condition.Status).WithMessage(condition.Message).WithReason(condition.Reason)
		}
	}
	return harborClusterCRUnknownStatus(EmptyHarborCRStatusError, "The ready condition of harbor.goharbor.io is empty. Please wait for minutes.")
}

func (harbor *HarborController) getHarborCRNamespacedName(harborcluster *v1alpha2.HarborCluster) types.NamespacedName {
	return types.NamespacedName{
		Namespace: harborcluster.Namespace,
		Name:      fmt.Sprintf("%s-harbor", harborcluster.Name),
	}
}
