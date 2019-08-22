package rediscluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1beta1 "github.com/ucloud/redis-operator/pkg/apis/redis/v1beta1"
)

// Ensure the RedisCluster's components are correct.
func (r *RedisClusterHandler) Ensure(rc *redisv1beta1.RedisCluster, labels map[string]string, or []metav1.OwnerReference) error {
	if rc.Spec.Exporter.Enabled {
		if err := r.rcService.EnsureRedisService(rc, labels, or); err != nil {
			return err
		}
	} else {
		if err := r.rcService.EnsureNotPresentRedisService(rc); err != nil {
			return err
		}
	}
	if err := r.rcService.EnsureSentinelService(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureSentinelConfigMap(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisShutdownConfigMap(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureRedisStatefulset(rc, labels, or); err != nil {
		return err
	}
	if err := r.rcService.EnsureSentinelDeployment(rc, labels, or); err != nil {
		return err
	}

	return nil
}
