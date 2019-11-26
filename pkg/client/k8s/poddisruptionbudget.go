package k8s

import (
	"context"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PodDisruptionBudget the client that knows how to interact with kubernetes to manage them
type PodDisruptionBudget interface {
	// GetPodDisruptionBudget get podDisruptionBudget from kubernetes with namespace and name
	GetPodDisruptionBudget(namespace string, name string) (*policyv1beta1.PodDisruptionBudget, error)
	// CreatePodDisruptionBudget will create the given podDisruptionBudget
	CreatePodDisruptionBudget(namespace string, podDisruptionBudget *policyv1beta1.PodDisruptionBudget) error
	// UpdatePodDisruptionBudget will update the given podDisruptionBudget
	UpdatePodDisruptionBudget(namespace string, podDisruptionBudget *policyv1beta1.PodDisruptionBudget) error
	// CreateOrUpdatePodDisruptionBudget will update the given podDisruptionBudget or create it if does not exist
	CreateOrUpdatePodDisruptionBudget(namespace string, podDisruptionBudget *policyv1beta1.PodDisruptionBudget) error
	// DeletePodDisruptionBudget will delete the given podDisruptionBudget
	DeletePodDisruptionBudget(namespace string, name string) error
	CreateIfNotExistsPodDisruptionBudget(namespace string, podDisruptionBudget *policyv1beta1.PodDisruptionBudget) error
}

// PodDisruptionBudgetOption is the podDisruptionBudget client implementation using API calls to kubernetes.
type PodDisruptionBudgetOption struct {
	client client.Client
	logger logr.Logger
}

// NewPodDisruptionBudget returns a new PodDisruptionBudget client.
func NewPodDisruptionBudget(kubeClient client.Client, logger logr.Logger) PodDisruptionBudget {
	logger = logger.WithValues("service", "k8s.podDisruptionBudget")
	return &PodDisruptionBudgetOption{
		client: kubeClient,
		logger: logger,
	}
}

// GetPodDisruptionBudget implement the PodDisruptionBudget.Interface
func (p *PodDisruptionBudgetOption) GetPodDisruptionBudget(namespace string, name string) (*policyv1beta1.PodDisruptionBudget, error) {
	podDisruptionBudget := &policyv1beta1.PodDisruptionBudget{}
	err := p.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, podDisruptionBudget)
	if err != nil {
		return nil, err
	}
	return podDisruptionBudget, nil
}

// CreatePodDisruptionBudget implement the PodDisruptionBudget.Interface
func (p *PodDisruptionBudgetOption) CreatePodDisruptionBudget(namespace string, podDisruptionBudget *policyv1beta1.PodDisruptionBudget) error {
	err := p.client.Create(context.TODO(), podDisruptionBudget)
	if err != nil {
		return err
	}
	p.logger.WithValues("namespace", namespace, "podDisruptionBudget", podDisruptionBudget.Name).Info("podDisruptionBudget created")
	return nil
}

// UpdatePodDisruptionBudget implement the PodDisruptionBudget.Interface
func (p *PodDisruptionBudgetOption) UpdatePodDisruptionBudget(namespace string, podDisruptionBudget *policyv1beta1.PodDisruptionBudget) error {
	err := p.client.Update(context.TODO(), podDisruptionBudget)
	if err != nil {
		return err
	}
	p.logger.WithValues("namespace", namespace, "podDisruptionBudget", podDisruptionBudget.Name).Info("podDisruptionBudget updated")
	return nil
}

// CreateOrUpdatePodDisruptionBudget implement the PodDisruptionBudget.Interface
func (p *PodDisruptionBudgetOption) CreateOrUpdatePodDisruptionBudget(namespace string, podDisruptionBudget *policyv1beta1.PodDisruptionBudget) error {
	storedPodDisruptionBudget, err := p.GetPodDisruptionBudget(namespace, podDisruptionBudget.Name)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return p.CreatePodDisruptionBudget(namespace, podDisruptionBudget)
		}
		return err
	}

	// Already exists, need to Update.
	// Set the correct resource version to ensure we are on the latest version. This way the only valid
	// namespace is our spec(https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency),
	// we will replace the current namespace state.
	podDisruptionBudget.ResourceVersion = storedPodDisruptionBudget.ResourceVersion
	return p.UpdatePodDisruptionBudget(namespace, podDisruptionBudget)
}

// CreateIfNotExistsPodDisruptionBudget implement the PodDisruptionBudget.Interface
func (p *PodDisruptionBudgetOption) CreateIfNotExistsPodDisruptionBudget(namespace string, podDisruptionBudget *policyv1beta1.PodDisruptionBudget) error {
	_, err := p.GetPodDisruptionBudget(namespace, podDisruptionBudget.Name)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return p.CreatePodDisruptionBudget(namespace, podDisruptionBudget)
		}
		return err
	}

	return nil
}

// DeletePodDisruptionBudget implement the PodDisruptionBudget.Interface
func (p *PodDisruptionBudgetOption) DeletePodDisruptionBudget(namespace string, name string) error {
	podDisruptionBudget := &policyv1beta1.PodDisruptionBudget{}
	if err := p.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, podDisruptionBudget); err != nil {
		return err
	}
	return p.client.Delete(context.TODO(), podDisruptionBudget)
}
