package k8s_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubetesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/ucloud/redis-operator/pkg/client/k8s"
	"github.com/ucloud/redis-operator/test/client"
)

var (
	podDisruptionBudgetsGroup = schema.GroupVersionResource{Group: "policy", Version: "v1beta1", Resource: "poddisruptionbudgets"}
)

func newPodDisruptionBudgetUpdateAction(ns string, podDisruptionBudget *policyv1beta1.PodDisruptionBudget) kubetesting.UpdateActionImpl {
	return kubetesting.NewUpdateAction(podDisruptionBudgetsGroup, ns, podDisruptionBudget)
}

func newPodDisruptionBudgetGetAction(ns, name string) kubetesting.GetActionImpl {
	return kubetesting.NewGetAction(podDisruptionBudgetsGroup, ns, name)
}

func newPodDisruptionBudgetCreateAction(ns string, podDisruptionBudget *policyv1beta1.PodDisruptionBudget) kubetesting.CreateActionImpl {
	return kubetesting.NewCreateAction(podDisruptionBudgetsGroup, ns, podDisruptionBudget)
}

func TestPodDisruptionBudgetServiceGetCreateOrUpdate(t *testing.T) {
	testPodDisruptionBudget := &policyv1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testpodDisruptionBudget1",
			ResourceVersion: "10",
		},
	}

	testns := "testns"

	tests := []struct {
		name                         string
		podDisruptionBudget          *policyv1beta1.PodDisruptionBudget
		getPodDisruptionBudgetResult *policyv1beta1.PodDisruptionBudget
		errorOnGet                   error
		errorOnCreation              error
		expActions                   []kubetesting.Action
		expErr                       bool
	}{
		{
			name:                         "A new podDisruptionBudget should create a new podDisruptionBudget.",
			podDisruptionBudget:          testPodDisruptionBudget,
			getPodDisruptionBudgetResult: nil,
			errorOnGet:                   kubeerrors.NewNotFound(schema.GroupResource{}, ""),
			errorOnCreation:              nil,
			expActions: []kubetesting.Action{
				newPodDisruptionBudgetGetAction(testns, testPodDisruptionBudget.ObjectMeta.Name),
				newPodDisruptionBudgetCreateAction(testns, testPodDisruptionBudget),
			},
			expErr: false,
		},
		{
			name:                         "A new podDisruptionBudget should error when create a new podDisruptionBudget fails.",
			podDisruptionBudget:          testPodDisruptionBudget,
			getPodDisruptionBudgetResult: nil,
			errorOnGet:                   kubeerrors.NewNotFound(schema.GroupResource{}, ""),
			errorOnCreation:              errors.New("wanted error"),
			expActions: []kubetesting.Action{
				newPodDisruptionBudgetGetAction(testns, testPodDisruptionBudget.ObjectMeta.Name),
				newPodDisruptionBudgetCreateAction(testns, testPodDisruptionBudget),
			},
			expErr: true,
		},
		{
			name:                         "An existent podDisruptionBudget should update the podDisruptionBudget.",
			podDisruptionBudget:          testPodDisruptionBudget,
			getPodDisruptionBudgetResult: testPodDisruptionBudget,
			errorOnGet:                   nil,
			errorOnCreation:              nil,
			expActions: []kubetesting.Action{
				newPodDisruptionBudgetGetAction(testns, testPodDisruptionBudget.ObjectMeta.Name),
				newPodDisruptionBudgetUpdateAction(testns, testPodDisruptionBudget),
			},
			expErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			cfg, err := config.GetConfig()
			if err != nil {
				panic(err)
			}
			kubeClient, err := client.NewK8sClient(cfg)
			if err != nil {
				panic(err)
			}

			service := k8s.NewPodDisruptionBudget(kubeClient, log)
			err = service.CreateOrUpdatePodDisruptionBudget(testns, test.podDisruptionBudget)

			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
