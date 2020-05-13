package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RedisClusterSpec defines the desired state of RedisCluster
// +k8s:openapi-gen=true
type RedisClusterSpec struct {
	Size               int32                         `json:"size,omitempty"`
	Resources          corev1.ResourceRequirements   `json:"resources,omitempty"`
	Image              string                        `json:"image,omitempty"`
	ImagePullPolicy    corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets   []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Command            []string                      `json:"command,omitempty"`
	ShutdownConfigMap  string                        `json:"shutdownConfigMap,omitempty"`
	Storage            RedisStorage                  `json:"storage,omitempty"`
	Password           string                        `json:"password,omitempty"`
	Exporter           RedisExporter                 `json:"exporter,omitempty"`
	Affinity           *corev1.Affinity              `json:"affinity,omitempty"`
	SecurityContext    *corev1.PodSecurityContext    `json:"securityContext,omitempty"`
	ToleRations        []corev1.Toleration           `json:"toleRations,omitempty"`
	NodeSelector       map[string]string             `json:"nodeSelector,omitempty"`
	Config             map[string]string             `json:"config,omitempty"`
	Annotations        map[string]string             `json:"annotations,omitempty"`
	DisablePersistence bool                          `json:"disablePersistence,omitempty"`

	// Sentinel defines its cluster settings
	Sentinel SentinelSettings `json:"sentinel,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RedisCluster is the Schema for the redisclusters API
// +k8s:openapi-gen=true
type RedisCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisClusterSpec   `json:"spec,omitempty"`
	Status RedisClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RedisClusterList contains a list of RedisCluster
type RedisClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisCluster{}, &RedisClusterList{})
}

// RedisExporter defines the specification for the redis exporter
type RedisExporter struct {
	Enabled         bool              `json:"enabled,omitempty"`
	Image           string            `json:"image,omitempty"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// SentinelSettings defines the specification of the sentinel cluster
type SentinelSettings struct {
	Image            string                        `json:"image,omitempty"`
	ImagePullPolicy  corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Replicas         int32                         `json:"replicas,omitempty"`
	Resources        corev1.ResourceRequirements   `json:"resources,omitempty"`
	CustomConfig     []string                      `json:"customConfig,omitempty"`
	Command          []string                      `json:"command,omitempty"`
	Affinity         *corev1.Affinity              `json:"affinity,omitempty"`
	SecurityContext  *corev1.PodSecurityContext    `json:"securityContext,omitempty"`
	ToleRations      []corev1.Toleration           `json:"tolerations,omitempty"`
	NodeSelector     map[string]string             `json:"nodeSelector,omitempty"`
	Annotations      map[string]string             `json:"annotations,omitempty"`
}

// RedisStorage defines the structure used to store the Redis Data
type RedisStorage struct {
	KeepAfterDeletion     bool                          `json:"keepAfterDeletion,omitempty"`
	EmptyDir              *corev1.EmptyDirVolumeSource  `json:"emptyDir,omitempty"`
	PersistentVolumeClaim *corev1.PersistentVolumeClaim `json:"persistentVolumeClaim,omitempty"`
}
