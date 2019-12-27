package k8s

import (
	"context"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Deployment the client that knows how to interact with kubernetes to manage them
type Deployment interface {
	// GetDeployment get deployment from kubernetes with namespace and name
	GetDeployment(namespace, name string) (*appsv1.Deployment, error)
	// GetDeploymentPods will retrieve the pods managed by a given deployment
	GetDeploymentPods(namespace, name string) (*corev1.PodList, error)
	// CreateDeployment will create the given deployment
	CreateDeployment(namespace string, deployment *appsv1.Deployment) error
	// UpdateDeployment will update the given deployment
	UpdateDeployment(namespace string, deployment *appsv1.Deployment) error
	// CreateOrUpdateDeployment will update the given deployment or create it if does not exist
	CreateOrUpdateDeployment(namespace string, deployment *appsv1.Deployment) error
	// DeleteDeployment will delete the given deployment
	DeleteDeployment(namespace string, name string) error
	// ListDeployments get set of deployment on a given namespace
	ListDeployments(namespace string) (*appsv1.DeploymentList, error)
}

// DeploymentOption is the deployment client interface implementation that using API calls to kubernetes.
type DeploymentOption struct {
	client client.Client
	logger logr.Logger
}

// NewDeployment returns a new Deployment client.
func NewDeployment(kubeClient client.Client, logger logr.Logger) Deployment {
	logger = logger.WithValues("service", "k8s.deployment")
	return &DeploymentOption{
		client: kubeClient,
		logger: logger,
	}
}

// GetDeployment implement the Deployment.Interface
func (d *DeploymentOption) GetDeployment(namespace, name string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := d.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, deployment)
	if err != nil {
		return nil, err
	}
	return deployment, err
}

// GetDeploymentPods implement the Deployment.Interface
func (d *DeploymentOption) GetDeploymentPods(namespace, name string) (*corev1.PodList, error) {
	deployment, err := d.GetDeployment(namespace, name)
	if err != nil {
		return nil, err
	}
	labelSet := make(map[string]string)
	for k, v := range deployment.Spec.Selector.MatchLabels {
		labelSet[k] = v
	}
	labelSelector := labels.SelectorFromSet(labelSet)
	foundPods := &corev1.PodList{}
	err = d.client.List(context.TODO(), foundPods, &client.ListOptions{Namespace: namespace, LabelSelector: labelSelector})
	return foundPods, err
}

// CreateDeployment implement the Deployment.Interface
func (d *DeploymentOption) CreateDeployment(namespace string, deployment *appsv1.Deployment) error {
	err := d.client.Create(context.TODO(), deployment)
	if err != nil {
		return err
	}
	d.logger.WithValues("namespace", namespace, "deployment", deployment.ObjectMeta.Name).Info("deployment created")
	return err
}

// UpdateDeployment implement the Deployment.Interface
func (d *DeploymentOption) UpdateDeployment(namespace string, deployment *appsv1.Deployment) error {
	err := d.client.Update(context.TODO(), deployment)
	if err != nil {
		return err
	}
	d.logger.WithValues("namespace", namespace, "deployment", deployment.ObjectMeta.Name).Info("deployment updated")
	return err
}

// CreateOrUpdateDeployment implement the Deployment.Interface
func (d *DeploymentOption) CreateOrUpdateDeployment(namespace string, deployment *appsv1.Deployment) error {
	storedDeployment, err := d.GetDeployment(namespace, deployment.Name)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return d.CreateDeployment(namespace, deployment)
		}
		return err
	}

	// Already exists, need to Update.
	// Set the correct resource version to ensure we are on the latest version. This way the only valid
	// namespace is our spec(https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency),
	// we will replace the current namespace state.
	deployment.ResourceVersion = storedDeployment.ResourceVersion
	return d.UpdateDeployment(namespace, deployment)
}

// DeleteDeployment implement the Deployment.Interface
func (d *DeploymentOption) DeleteDeployment(namespace, name string) error {
	deployment := &appsv1.Deployment{}
	if err := d.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, deployment); err != nil {
		return err
	}
	return d.client.Delete(context.TODO(), deployment)
}

// ListDeployments implement the Deployment.Interface
func (d *DeploymentOption) ListDeployments(namespace string) (*appsv1.DeploymentList, error) {
	ds := &appsv1.DeploymentList{}
	listOps := &client.ListOptions{
		Namespace: namespace,
	}
	err := d.client.List(context.TODO(), ds, listOps)
	return ds, err
}
