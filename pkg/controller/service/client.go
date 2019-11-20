package service

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	redisv1beta1 "github.com/ucloud/redis-operator/pkg/apis/redis/v1beta1"
	"github.com/ucloud/redis-operator/pkg/client/k8s"
	"github.com/ucloud/redis-operator/pkg/util"
)

// RedisClusterClient has the minimumm methods that a Redis cluster controller needs to satisfy
// in order to talk with K8s
type RedisClusterClient interface {
	EnsureSentinelService(redisCluster *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureSentinelHeadlessService(redisCluster *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureSentinelConfigMap(redisCluster *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureSentinelProbeConfigMap(redisCluster *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureSentinelStatefulset(redisCluster *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisStatefulset(redisCluster *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisService(redisCluster *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisShutdownConfigMap(redisCluster *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureRedisConfigMap(redisCluster *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error
	EnsureNotPresentRedisService(redisCluster *redisv1beta1.RedisCluster) error
}

// RedisClusterKubeClient implements the required methods to talk with kubernetes
type RedisClusterKubeClient struct {
	K8SService k8s.Services
	logger     logr.Logger
}

// NewRedisClusterKubeClient creates a new RedisClusterKubeClient
func NewRedisClusterKubeClient(k8sService k8s.Services, logger logr.Logger) *RedisClusterKubeClient {
	return &RedisClusterKubeClient{
		K8SService: k8sService,
		logger:     logger,
	}
}

func generateSelectorLabels(component, name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/part-of":   util.AppLabel,
		"app.kubernetes.io/component": component,
		"app.kubernetes.io/name":      name,
	}
}

// EnsureSentinelService makes sure the sentinel service exists
func (r *RedisClusterKubeClient) EnsureSentinelService(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := generateSentinelService(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsService(rc.Namespace, svc)
}

// EnsureSentinelHeadlessService makes sure the sentinel headless service exists
func (r *RedisClusterKubeClient) EnsureSentinelHeadlessService(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := newHeadLessSvcForCR(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsService(rc.Namespace, svc)
}

// EnsureSentinelConfigMap makes sure the sentinel configmap exists
func (r *RedisClusterKubeClient) EnsureSentinelConfigMap(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	cm := generateSentinelConfigMap(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsConfigMap(rc.Namespace, cm)
}

// EnsureSentinelConfigMap makes sure the sentinel configmap exists
func (r *RedisClusterKubeClient) EnsureSentinelProbeConfigMap(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	cm := generateSentinelReadinessProbeConfigMap(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsConfigMap(rc.Namespace, cm)
}

// EnsureSentinelStatefulset makes sure the sentinel deployment exists in the desired state
func (r *RedisClusterKubeClient) EnsureSentinelStatefulset(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	if err := r.ensurePodDisruptionBudget(rc, util.SentinelName, util.SentinelRoleName, labels, ownerRefs); err != nil {
		return err
	}

	oldSs, err := r.K8SService.GetStatefulSet(rc.Namespace, util.GetSentinelName(rc))
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			ss := generateSentinelStatefulSet(rc, labels, ownerRefs)
			return r.K8SService.CreateStatefulSet(rc.Namespace, ss)
		}
		return err
	}

	if shouldUpdateRedis(rc, oldSs.Spec.Template.Spec.Containers[0], *oldSs.Spec.Replicas) {
		ss := generateSentinelStatefulSet(rc, labels, ownerRefs)
		return r.K8SService.UpdateStatefulSet(rc.Namespace, ss)
	}
	return nil
}

// EnsureRedisStatefulset makes sure the redis statefulset exists in the desired state
func (r *RedisClusterKubeClient) EnsureRedisStatefulset(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	if err := r.ensurePodDisruptionBudget(rc, util.RedisName, util.RedisRoleName, labels, ownerRefs); err != nil {
		return err
	}

	ns, err := r.K8SService.GetNameSpace(rc.Namespace)
	//r.logger.WithValues("namespace", rc.Namespace).Info("get namespace", ns, err)
	if err != nil {
		return err
	}
	annotationIstioInject := false
	if v, ok := ns.Labels["istio-injection"]; ok && v == "enabled" {
		annotationIstioInject = true
	}

	oldSs, err := r.K8SService.GetStatefulSet(rc.Namespace, util.GetRedisName(rc))
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			ss := generateRedisStatefulSet(rc, labels, ownerRefs, annotationIstioInject)
			return r.K8SService.CreateStatefulSet(rc.Namespace, ss)
		}
		return err
	}

	if shouldUpdateRedis(rc, oldSs.Spec.Template.Spec.Containers[0], *oldSs.Spec.Replicas) {
		ss := generateRedisStatefulSet(rc, labels, ownerRefs, annotationIstioInject)
		return r.K8SService.UpdateStatefulSet(rc.Namespace, ss)
	}
	return nil
}

func shouldUpdateRedis(rc *redisv1beta1.RedisCluster, containter corev1.Container, replicas int32) bool {
	if replicas != rc.Spec.Size {
		return true
	}
	if result := containter.Resources.Requests.Cpu().Cmp(*rc.Spec.Resources.Requests.Cpu()); result != 0 {
		return true
	}
	if result := containter.Resources.Requests.Memory().Cmp(*rc.Spec.Resources.Requests.Memory()); result != 0 {
		return true
	}
	if result := containter.Resources.Limits.Cpu().Cmp(*rc.Spec.Resources.Limits.Cpu()); result != 0 {
		return true
	}
	if result := containter.Resources.Limits.Memory().Cmp(*rc.Spec.Resources.Limits.Memory()); result != 0 {
		return true
	}
	return false
}

// EnsureRedisConfigMap makes sure the sentinel configmap exists
func (r *RedisClusterKubeClient) EnsureRedisConfigMap(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	cm := generateRedisConfigMap(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsConfigMap(rc.Namespace, cm)
}

// EnsureRedisShutdownConfigMap makes sure the redis configmap with shutdown script exists
func (r *RedisClusterKubeClient) EnsureRedisShutdownConfigMap(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	if rc.Spec.ShutdownConfigMap != "" {
		if _, err := r.K8SService.GetConfigMap(rc.Namespace, rc.Spec.ShutdownConfigMap); err != nil {
			return err
		}
	} else {
		cm := generateRedisShutdownConfigMap(rc, labels, ownerRefs)
		return r.K8SService.CreateIfNotExistsConfigMap(rc.Namespace, cm)
	}
	return nil
}

// EnsureRedisService makes sure the redis statefulset exists
func (r *RedisClusterKubeClient) EnsureRedisService(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	svc := generateRedisService(rc, labels, ownerRefs)
	return r.K8SService.CreateIfNotExistsService(rc.Namespace, svc)
}

// EnsureNotPresentRedisService makes sure the redis service is not present
func (r *RedisClusterKubeClient) EnsureNotPresentRedisService(rc *redisv1beta1.RedisCluster) error {
	name := util.GetRedisName(rc)
	namespace := rc.Namespace
	// If the service exists (no get error), delete it
	if _, err := r.K8SService.GetService(namespace, name); err == nil {
		return r.K8SService.DeleteService(namespace, name)
	}
	return nil
}

// EnsureRedisStatefulset makes sure the pdb exists in the desired state
func (r *RedisClusterKubeClient) ensurePodDisruptionBudget(rc *redisv1beta1.RedisCluster, name string, component string, labels map[string]string, ownerRefs []metav1.OwnerReference) error {
	name = util.GenerateName(name, rc.Name)
	namespace := rc.Namespace

	minAvailable := intstr.FromInt(2)
	labels = util.MergeLabels(labels, generateSelectorLabels(component, rc.Name))

	pdb := generatePodDisruptionBudget(name, namespace, labels, ownerRefs, minAvailable)

	return r.K8SService.CreateIfNotExistsPodDisruptionBudget(namespace, pdb)
}
