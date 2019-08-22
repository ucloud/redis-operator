package k8s_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubetesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/ucloud/redis-operator/pkg/client/k8s"
	"github.com/ucloud/redis-operator/test/client"
)

var (
	statefulSetsGroup = schema.GroupVersionResource{Group: "apps", Version: "v1beta2", Resource: "statefulsets"}
)

func newStatefulSetUpdateAction(ns string, statefulSet *appsv1.StatefulSet) kubetesting.UpdateActionImpl {
	return kubetesting.NewUpdateAction(statefulSetsGroup, ns, statefulSet)
}

func newStatefulSetGetAction(ns, name string) kubetesting.GetActionImpl {
	return kubetesting.NewGetAction(statefulSetsGroup, ns, name)
}

func newStatefulSetCreateAction(ns string, statefulSet *appsv1.StatefulSet) kubetesting.CreateActionImpl {
	return kubetesting.NewCreateAction(statefulSetsGroup, ns, statefulSet)
}

func TestStatefulSetServiceGetCreateOrUpdate(t *testing.T) {
	testStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "teststatefulSet1",
			ResourceVersion: "10",
		},
	}

	testns := "testns"

	tests := []struct {
		name                 string
		statefulSet          *appsv1.StatefulSet
		getStatefulSetResult *appsv1.StatefulSet
		errorOnGet           error
		errorOnCreation      error
		expActions           []kubetesting.Action
		expErr               bool
	}{
		{
			name:                 "A new statefulSet should create a new statefulSet.",
			statefulSet:          testStatefulSet,
			getStatefulSetResult: nil,
			errorOnGet:           kubeerrors.NewNotFound(schema.GroupResource{}, ""),
			errorOnCreation:      nil,
			expActions: []kubetesting.Action{
				newStatefulSetGetAction(testns, testStatefulSet.ObjectMeta.Name),
				newStatefulSetCreateAction(testns, testStatefulSet),
			},
			expErr: false,
		},
		{
			name:                 "A new statefulSet should error when create a new statefulSet fails.",
			statefulSet:          testStatefulSet,
			getStatefulSetResult: nil,
			errorOnGet:           kubeerrors.NewNotFound(schema.GroupResource{}, ""),
			errorOnCreation:      errors.New("wanted error"),
			expActions: []kubetesting.Action{
				newStatefulSetGetAction(testns, testStatefulSet.ObjectMeta.Name),
				newStatefulSetCreateAction(testns, testStatefulSet),
			},
			expErr: true,
		},
		{
			name:                 "An existent statefulSet should update the statefulSet.",
			statefulSet:          testStatefulSet,
			getStatefulSetResult: testStatefulSet,
			errorOnGet:           nil,
			errorOnCreation:      nil,
			expActions: []kubetesting.Action{
				newStatefulSetGetAction(testns, testStatefulSet.ObjectMeta.Name),
				newStatefulSetUpdateAction(testns, testStatefulSet),
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

			service := k8s.NewStatefulSet(kubeClient, log)
			err = service.CreateOrUpdateStatefulSet(testns, test.statefulSet)

			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
