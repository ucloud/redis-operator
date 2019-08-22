package k8s

import (
	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Service is the kubernetes service entrypoint.
type Services interface {
	ConfigMap
	Pod
	PodDisruptionBudget
	Service
	NameSpaces
	Deployment
	StatefulSet
	Cluster
}

type services struct {
	ConfigMap
	Pod
	PodDisruptionBudget
	Service
	NameSpaces
	Deployment
	StatefulSet
	Cluster
}

// New returns a new Kubernetes client set.
func New(kubecli client.Client, logger logr.Logger) Services {
	return &services{
		ConfigMap:           NewConfigMap(kubecli, logger),
		Pod:                 NewPod(kubecli, logger),
		PodDisruptionBudget: NewPodDisruptionBudget(kubecli, logger),
		Service:             NewService(kubecli, logger),
		NameSpaces:          NewNameSpaces(logger),
		Deployment:          NewDeployment(kubecli, logger),
		StatefulSet:         NewStatefulSet(kubecli, logger),
		Cluster:             NewCluster(kubecli, logger),
	}
}
