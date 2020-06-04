package service

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	redisv1beta1 "github.com/ucloud/redis-operator/pkg/apis/redis/v1beta1"
	"github.com/ucloud/redis-operator/pkg/util"
)

const (
	redisShutdownConfigurationVolumeName = "redis-shutdown-config"
	redisStorageVolumeName               = "redis-data"

	graceTime = 30
)

func generateSentinelService(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	name := util.GetSentinelName(rc)
	namespace := rc.Namespace

	sentinelTargetPort := intstr.FromInt(26379)
	labels = util.MergeLabels(labels, generateSelectorLabels(util.SentinelRoleName, rc.Name))

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "sentinel",
					Port:       26379,
					TargetPort: sentinelTargetPort,
					Protocol:   "TCP",
				},
			},
		},
	}
}

func generateRedisService(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	name := util.GetRedisName(rc)
	namespace := rc.Namespace

	labels = util.MergeLabels(labels, generateSelectorLabels(util.RedisRoleName, rc.Name))
	redisTargetPort := intstr.FromInt(6379)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{
				{
					Port:       6379,
					Protocol:   corev1.ProtocolTCP,
					Name:       "redis",
					TargetPort: redisTargetPort,
				},
			},
			Selector: labels,
		},
	}
}

func generateSentinelConfigMap(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	name := util.GetSentinelName(rc)
	namespace := rc.Namespace

	labels = util.MergeLabels(labels, generateSelectorLabels(util.SentinelRoleName, rc.Name))
	sentinelConfigFileContent := `sentinel monitor mymaster 127.0.0.1 6379 2
sentinel down-after-milliseconds mymaster 1000
sentinel failover-timeout mymaster 3000
sentinel parallel-syncs mymaster 2`

	if rc.Spec.Password != "" {
		sentinelConfigFileContent = fmt.Sprintf("%s\nsentinel auth-pass mymaster %s\n", sentinelConfigFileContent, rc.Spec.Password)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{
			util.SentinelConfigFileName: sentinelConfigFileContent,
		},
	}
}

func generateRedisConfigMap(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	name := util.GetRedisName(rc)
	namespace := rc.Namespace

	labels = util.MergeLabels(labels, generateSelectorLabels(util.RedisRoleName, rc.Name))
	redisConfigFileContent := `slaveof 127.0.0.1 6379
tcp-keepalive 60
save 900 1
save 300 10`
	if rc.Spec.Password != "" {
		redisConfigFileContent = fmt.Sprintf("%s\nrequirepass %s\nmasterauth %s\n", redisConfigFileContent, rc.Spec.Password, rc.Spec.Password)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{
			util.RedisConfigFileName: redisConfigFileContent,
		},
	}
}

func generateRedisShutdownConfigMap(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	name := util.GetRedisShutdownConfigMapName(rc)
	namespace := rc.Namespace

	labels = util.MergeLabels(labels, generateSelectorLabels(util.RedisRoleName, rc.Name))
	envSentinelHost := fmt.Sprintf("REDIS_SENTINEL_%s_SERVICE_HOST", strings.ToUpper(rc.Name))
	envSentinelPort := fmt.Sprintf("REDIS_SENTINEL_%s_SERVICE_PORT_SENTINEL", strings.ToUpper(rc.Name))
	shutdownContent := fmt.Sprintf(`#!/usr/bin/env sh
set -eou pipefail
master=$(redis-cli -h ${%s} -p ${%s} --csv SENTINEL get-master-addr-by-name mymaster | tr ',' ' ' | tr -d '\"' |cut -d' ' -f1)
redis-cli SAVE
if [[ $master ==  $(hostname -i) ]]; then
  redis-cli -h ${%s} -p ${%s} SENTINEL failover mymaster
fi`, envSentinelHost, envSentinelPort, envSentinelHost, envSentinelPort)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{
			"shutdown.sh": shutdownContent,
		},
	}
}

func generateSentinelReadinessProbeConfigMap(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.ConfigMap {
	name := util.GetSentinelReadinessCm(rc)
	namespace := rc.Namespace

	labels = util.MergeLabels(labels, generateSelectorLabels(util.RedisRoleName, rc.Name))
	checkContent := `#!/usr/bin/env sh
set -eou pipefail
redis-cli -h $(hostname) -p 26379 ping
slaves=$(redis-cli -h $(hostname) -p 26379 info sentinel|grep master0| grep -Eo 'slaves=[0-9]+' | awk -F= '{print $2}')
status=$(redis-cli -h $(hostname) -p 26379 info sentinel|grep master0| grep -Eo 'status=\w+' | awk -F= '{print $2}')
if [ "$status" != "ok" ]; then 
    exit 1
fi
if [ $slaves -le 1 ]; then
	exit 1
fi
`

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Data: map[string]string{
			"readiness.sh": checkContent,
		},
	}
}

func generateRedisStatefulSet(rc *redisv1beta1.RedisCluster, labels map[string]string,
	ownerRefs []metav1.OwnerReference) *appsv1.StatefulSet {
	name := util.GetRedisName(rc)
	namespace := rc.Namespace

	spec := rc.Spec
	redisCommand := getRedisCommand(rc)
	labels = util.MergeLabels(labels, generateSelectorLabels(util.RedisRoleName, rc.Name))
	volumeMounts := getRedisVolumeMounts(rc)
	volumes := getRedisVolumes(rc)

	probeArg := "redis-cli -h $(hostname)"
	if spec.Password != "" {
		probeArg = fmt.Sprintf("%s -a '%s' ping", probeArg, spec.Password)
	} else {
		probeArg = fmt.Sprintf("%s ping", probeArg)
	}

	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name,
			Replicas:    &spec.Size,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: "RollingUpdate",
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: rc.Spec.Annotations,
				},
				Spec: corev1.PodSpec{
					Affinity:         getAffinity(rc.Spec.Affinity, labels),
					Tolerations:      rc.Spec.ToleRations,
					NodeSelector:     rc.Spec.NodeSelector,
					SecurityContext:  getSecurityContext(rc.Spec.SecurityContext),
					ImagePullSecrets: rc.Spec.ImagePullSecrets,
					Containers: []corev1.Container{
						{
							Name:            "redis",
							Image:           rc.Spec.Image,
							ImagePullPolicy: pullPolicy(rc.Spec.ImagePullPolicy),
							Ports: []corev1.ContainerPort{
								{
									Name:          "redis",
									ContainerPort: 6379,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: volumeMounts,
							Command:      redisCommand,
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: graceTime,
								TimeoutSeconds:      5,
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"sh",
											"-c",
											probeArg,
										},
									},
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: graceTime,
								TimeoutSeconds:      5,
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"sh",
											"-c",
											probeArg,
										},
									},
								},
							},
							Resources: rc.Spec.Resources,
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/sh", "/redis-shutdown/shutdown.sh"},
									},
								},
							},
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	if rc.Spec.Storage.PersistentVolumeClaim != nil {
		if !rc.Spec.Storage.KeepAfterDeletion {
			// Set an owner reference so the persistent volumes are deleted when the rc is
			rc.Spec.Storage.PersistentVolumeClaim.OwnerReferences = ownerRefs
		}
		ss.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			*rc.Spec.Storage.PersistentVolumeClaim,
		}
	}

	if rc.Spec.Exporter.Enabled {
		exporter := createRedisExporterContainer(rc)
		ss.Spec.Template.Spec.Containers = append(ss.Spec.Template.Spec.Containers, exporter)
	}

	return ss
}

func generateSentinelStatefulSet(rc *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) *appsv1.StatefulSet {
	name := util.GetSentinelName(rc)
	configMapName := util.GetSentinelName(rc)
	namespace := rc.Namespace

	spec := rc.Spec
	sentinelCommand := getSentinelCommand(rc)
	labels = util.MergeLabels(labels, generateSelectorLabels(util.SentinelRoleName, rc.Name))

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: util.GetSentinelHeadlessSvc(rc),
			Replicas:    &spec.Sentinel.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: rc.Spec.Sentinel.Annotations,
				},
				Spec: corev1.PodSpec{
					Affinity:         getAffinity(rc.Spec.Sentinel.Affinity, labels),
					Tolerations:      rc.Spec.Sentinel.ToleRations,
					NodeSelector:     rc.Spec.Sentinel.NodeSelector,
					SecurityContext:  getSecurityContext(rc.Spec.Sentinel.SecurityContext),
					ImagePullSecrets: rc.Spec.Sentinel.ImagePullSecrets,
					InitContainers: []corev1.Container{
						{
							Name:            "sentinel-config-copy",
							Image:           rc.Spec.Sentinel.Image,
							ImagePullPolicy: pullPolicy(rc.Spec.Sentinel.ImagePullPolicy),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "sentinel-config",
									MountPath: "/redis",
								},
								{
									Name:      "sentinel-config-writable",
									MountPath: "/redis-writable",
								},
							},
							Command: []string{
								"cp",
								fmt.Sprintf("/redis/%s", util.SentinelConfigFileName),
								fmt.Sprintf("/redis-writable/%s", util.SentinelConfigFileName),
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("10m"),
									corev1.ResourceMemory: resource.MustParse("32Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("10m"),
									corev1.ResourceMemory: resource.MustParse("32Mi"),
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "sentinel",
							Image:           rc.Spec.Sentinel.Image,
							ImagePullPolicy: pullPolicy(rc.Spec.Sentinel.ImagePullPolicy),
							Ports: []corev1.ContainerPort{
								{
									Name:          "sentinel",
									ContainerPort: 26379,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "readiness-probe",
									MountPath: "/redis-probe",
								},
								{
									Name:      "sentinel-config-writable",
									MountPath: "/redis",
								},
							},
							Command: sentinelCommand,
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: graceTime,
								PeriodSeconds:       15,
								FailureThreshold:    5,
								TimeoutSeconds:      5,
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"sh",
											"/redis-probe/readiness.sh",
										},
									},
								},
							},
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: graceTime,
								TimeoutSeconds:      5,
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"sh",
											"-c",
											"redis-cli -h $(hostname) -p 26379 ping",
										},
									},
								},
							},
							Resources: rc.Spec.Sentinel.Resources,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "sentinel-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
						{
							Name: "readiness-probe",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: util.GetSentinelReadinessCm(rc),
									},
								},
							},
						},
						{
							Name: "sentinel-config-writable",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
}

func generatePodDisruptionBudget(name string, namespace string, labels map[string]string, ownerRefs []metav1.OwnerReference, minAvailable intstr.IntOrString) *policyv1beta1.PodDisruptionBudget {
	return &policyv1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}
}

func generateResourceList(cpu string, memory string) corev1.ResourceList {
	resources := corev1.ResourceList{}
	if cpu != "" {
		resources[corev1.ResourceCPU], _ = resource.ParseQuantity(cpu)
	}
	if memory != "" {
		resources[corev1.ResourceMemory], _ = resource.ParseQuantity(memory)
	}
	return resources
}

func createRedisExporterContainer(rc *redisv1beta1.RedisCluster) corev1.Container {
	container := corev1.Container{
		Name:            exporterContainerName,
		Image:           rc.Spec.Exporter.Image,
		ImagePullPolicy: pullPolicy(rc.Spec.Exporter.ImagePullPolicy),
		Env: []corev1.EnvVar{
			{
				Name: "REDIS_ALIAS",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          exporterPortName,
				ContainerPort: exporterPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(exporterDefaultLimitCPU),
				corev1.ResourceMemory: resource.MustParse(exporterDefaultLimitMemory),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(exporterDefaultRequestCPU),
				corev1.ResourceMemory: resource.MustParse(exporterDefaultRequestMemory),
			},
		},
	}
	if rc.Spec.Password != "" {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  redisPasswordEnv,
			Value: rc.Spec.Password,
		})
	}
	return container
}

func createPodAntiAffinity(hard bool, labels map[string]string) *corev1.PodAntiAffinity {
	if hard {
		// Return a HARD anti-affinity (no same pods on one node)
		return &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					TopologyKey: util.HostnameTopologyKey,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
				},
			},
		}
	}

	// Return a SOFT anti-affinity
	return &corev1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
			{
				Weight: 100,
				PodAffinityTerm: corev1.PodAffinityTerm{
					TopologyKey: util.HostnameTopologyKey,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
				},
			},
		},
	}
}

func getSecurityContext(secctx *corev1.PodSecurityContext) *corev1.PodSecurityContext {
	if secctx != nil {
		return secctx
	}

	return nil
}

func getQuorum(rc *redisv1beta1.RedisCluster) int32 {
	return rc.Spec.Sentinel.Replicas/2 + 1
}

func getRedisVolumeMounts(rc *redisv1beta1.RedisCluster) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		//{
		//	Name:      redisConfigurationVolumeName,
		//	MountPath: "/redis",
		//},
		{
			Name:      redisShutdownConfigurationVolumeName,
			MountPath: "/redis-shutdown",
		},
		{
			Name:      getRedisDataVolumeName(rc),
			MountPath: "/data",
		},
	}

	return volumeMounts
}

func getRedisVolumes(rc *redisv1beta1.RedisCluster) []corev1.Volume {
	shutdownConfigMapName := util.GetRedisShutdownConfigMapName(rc)

	executeMode := int32(0744)
	volumes := []corev1.Volume{
		{
			Name: redisShutdownConfigurationVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: shutdownConfigMapName,
					},
					DefaultMode: &executeMode,
				},
			},
		},
	}

	dataVolume := getRedisDataVolume(rc)
	if dataVolume != nil {
		volumes = append(volumes, *dataVolume)
	}

	return volumes
}

func getRedisDataVolume(rc *redisv1beta1.RedisCluster) *corev1.Volume {
	// This will find the volumed desired by the user. If no volume defined
	// an EmptyDir will be used by default
	switch {
	case rc.Spec.Storage.PersistentVolumeClaim != nil:
		return nil
	case rc.Spec.Storage.EmptyDir != nil:
		return &corev1.Volume{
			Name: redisStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: rc.Spec.Storage.EmptyDir,
			},
		}
	default:
		return &corev1.Volume{
			Name: redisStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
	}
}

func getRedisDataVolumeName(rc *redisv1beta1.RedisCluster) string {
	switch {
	case rc.Spec.Storage.PersistentVolumeClaim != nil:
		return rc.Spec.Storage.PersistentVolumeClaim.Name
	case rc.Spec.Storage.EmptyDir != nil:
		return redisStorageVolumeName
	default:
		return redisStorageVolumeName
	}
}

func getRedisCommand(rc *redisv1beta1.RedisCluster) []string {
	if len(rc.Spec.Command) > 0 {
		return rc.Spec.Command
	}

	cmds := []string{
		"redis-server",
		"--slaveof 127.0.0.1 6379",
		"--tcp-keepalive 60",
		"--save 900 1",
		"--save 300 10",
	}

	if rc.Spec.Password != "" {
		cmds = append(cmds, fmt.Sprintf("--requirepass '%s'", rc.Spec.Password),
			fmt.Sprintf("--masterauth '%s'", rc.Spec.Password))
	}

	return cmds
}

func getSentinelCommand(rc *redisv1beta1.RedisCluster) []string {
	if len(rc.Spec.Sentinel.Command) > 0 {
		return rc.Spec.Sentinel.Command
	}
	return []string{
		"redis-server",
		fmt.Sprintf("/redis/%s", util.SentinelConfigFileName),
		"--sentinel",
	}
}

func getAffinity(affinity *corev1.Affinity, labels map[string]string) *corev1.Affinity {
	if affinity != nil {
		return affinity
	}

	// Return a SOFT anti-affinity
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						TopologyKey: util.HostnameTopologyKey,
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: labels,
						},
					},
				},
			},
		},
	}
}

// newHeadLessSvcForCR creates a new headless service for the given Cluster.
func newHeadLessSvcForCR(cluster *redisv1beta1.RedisCluster, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Service {
	sentinelPort := corev1.ServicePort{Name: "sentinel", Port: 26379}
	labels = util.MergeLabels(labels, generateSelectorLabels(util.SentinelRoleName, cluster.Name))
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Name:            util.GetSentinelHeadlessSvc(cluster),
			Namespace:       cluster.Namespace,
			OwnerReferences: ownerRefs,
		},
		Spec: corev1.ServiceSpec{
			Ports:     []corev1.ServicePort{sentinelPort},
			Selector:  labels,
			ClusterIP: corev1.ClusterIPNone,
		},
	}

	return svc
}

func pullPolicy(specPolicy corev1.PullPolicy) corev1.PullPolicy {
	if specPolicy == "" {
		return corev1.PullAlways
	}
	return specPolicy
}
