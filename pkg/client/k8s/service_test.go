package k8s_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubetesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/ucloud/redis-operator/pkg/client/k8s"
	"github.com/ucloud/redis-operator/test/client"
)

var (
	servicesGroup = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
)

func newServiceUpdateAction(ns string, service *corev1.Service) kubetesting.UpdateActionImpl {
	return kubetesting.NewUpdateAction(servicesGroup, ns, service)
}

func newServiceGetAction(ns, name string) kubetesting.GetActionImpl {
	return kubetesting.NewGetAction(servicesGroup, ns, name)
}

func newServiceCreateAction(ns string, service *corev1.Service) kubetesting.CreateActionImpl {
	return kubetesting.NewCreateAction(servicesGroup, ns, service)
}

func TestServiceServiceGetCreateOrUpdate(t *testing.T) {
	testService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testservice1",
			ResourceVersion: "10",
		},
	}

	testns := "testns"

	tests := []struct {
		name             string
		service          *corev1.Service
		getServiceResult *corev1.Service
		errorOnGet       error
		errorOnCreation  error
		expActions       []kubetesting.Action
		expErr           bool
	}{
		{
			name:             "A new service should create a new service.",
			service:          testService,
			getServiceResult: nil,
			errorOnGet:       kubeerrors.NewNotFound(schema.GroupResource{}, ""),
			errorOnCreation:  nil,
			expActions: []kubetesting.Action{
				newServiceGetAction(testns, testService.ObjectMeta.Name),
				newServiceCreateAction(testns, testService),
			},
			expErr: false,
		},
		{
			name:             "A new service should error when create a new service fails.",
			service:          testService,
			getServiceResult: nil,
			errorOnGet:       kubeerrors.NewNotFound(schema.GroupResource{}, ""),
			errorOnCreation:  errors.New("wanted error"),
			expActions: []kubetesting.Action{
				newServiceGetAction(testns, testService.ObjectMeta.Name),
				newServiceCreateAction(testns, testService),
			},
			expErr: true,
		},
		{
			name:             "An existent service should update the service.",
			service:          testService,
			getServiceResult: testService,
			errorOnGet:       nil,
			errorOnCreation:  nil,
			expActions: []kubetesting.Action{
				newServiceGetAction(testns, testService.ObjectMeta.Name),
				newServiceUpdateAction(testns, testService),
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

			service := k8s.NewService(kubeClient, log)
			err = service.CreateOrUpdateService(testns, test.service)

			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
