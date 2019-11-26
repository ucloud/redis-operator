package util

import (
	"fmt"

	redisv1beta1 "github.com/ucloud/redis-operator/pkg/apis/redis/v1beta1"
)

const (
	BaseName               = "redis"
	SentinelName           = "-sentinel"
	SentinelRoleName       = "sentinel"
	SentinelConfigFileName = "sentinel.conf"
	RedisConfigFileName    = "redis.conf"
	RedisName              = "-cluster"
	RedisShutdownName      = "r-s"
	RedisRoleName          = "redis"
	AppLabel               = "redis-cluster"
	HostnameTopologyKey    = "kubernetes.io/hostname"
)

// GetRedisShutdownConfigMapName returns the name for redis configmap
func GetRedisShutdownConfigMapName(rc *redisv1beta1.RedisCluster) string {
	if rc.Spec.ShutdownConfigMap != "" {
		return rc.Spec.ShutdownConfigMap
	}
	return GetRedisShutdownName(rc)
}

// GetRedisName returns the name for redis resources
func GetRedisName(rc *redisv1beta1.RedisCluster) string {
	return GenerateName(RedisName, rc.Name)
}

// GetRedisShutdownName returns the name for redis resources
func GetRedisShutdownName(rc *redisv1beta1.RedisCluster) string {
	return GenerateName(RedisShutdownName, rc.Name)
}

// GetSentinelName returns the name for sentinel resources
func GetSentinelName(rc *redisv1beta1.RedisCluster) string {
	return GenerateName(SentinelName, rc.Name)
}

func GenerateName(typeName, metaName string) string {
	return fmt.Sprintf("%s%s-%s", BaseName, typeName, metaName)
}

func GetSentinelReadinessCm(rc *redisv1beta1.RedisCluster) string {
	return GenerateName("-sentinel-readiness", rc.Name)
}

func GetSentinelHeadlessSvc(rc *redisv1beta1.RedisCluster) string {
	return GenerateName("-sentinel-headless", rc.Name)
}
