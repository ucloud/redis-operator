package service

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/go-logr/logr"
	goredis "github.com/go-redis/redis"
	corev1 "k8s.io/api/core/v1"

	redisv1beta1 "github.com/ucloud/redis-operator/pkg/apis/redis/v1beta1"
	"github.com/ucloud/redis-operator/pkg/client/k8s"
	"github.com/ucloud/redis-operator/pkg/client/redis"
	"github.com/ucloud/redis-operator/pkg/util"
)

// RedisClusterCheck defines the intercace able to check the correct status of a redis cluster
type RedisClusterCheck interface {
	CheckRedisNumber(redisCluster *redisv1beta1.RedisCluster) error
	CheckSentinelNumber(redisCluster *redisv1beta1.RedisCluster) error
	CheckSentinelReadyReplicas(redisCluster *redisv1beta1.RedisCluster) error
	CheckAllSlavesFromMaster(master string, redisCluster *redisv1beta1.RedisCluster, auth *util.AuthConfig) error
	CheckSentinelNumberInMemory(sentinel string, redisCluster *redisv1beta1.RedisCluster, auth *util.AuthConfig) error
	CheckSentinelSlavesNumberInMemory(sentinel string, redisCluster *redisv1beta1.RedisCluster, auth *util.AuthConfig) error
	CheckSentinelMonitor(sentinel string, monitor string, auth *util.AuthConfig) error
	GetMasterIP(redisCluster *redisv1beta1.RedisCluster, auth *util.AuthConfig) (string, error)
	GetNumberMasters(redisCluster *redisv1beta1.RedisCluster, auth *util.AuthConfig) (int, error)
	GetRedisesIPs(redisCluster *redisv1beta1.RedisCluster, auth *util.AuthConfig) ([]string, error)
	GetSentinelsIPs(redisCluster *redisv1beta1.RedisCluster) ([]string, error)
	GetMinimumRedisPodTime(redisCluster *redisv1beta1.RedisCluster) (time.Duration, error)
	CheckRedisConfig(redisCluster *redisv1beta1.RedisCluster, addr string, auth *util.AuthConfig) error
}

var parseConfigMap = map[string]int8{
	"maxmemory":                  0,
	"proto-max-bulk-len":         0,
	"client-query-buffer-limit":  0,
	"repl-backlog-size":          0,
	"auto-aof-rewrite-min-size":  0,
	"active-defrag-ignore-bytes": 0,
	"hash-max-ziplist-entries":   0,
	"hash-max-ziplist-value":     0,
	"stream-node-max-bytes":      0,
	"set-max-intset-entries":     0,
	"zset-max-ziplist-entries":   0,
	"zset-max-ziplist-value":     0,
	"hll-sparse-max-bytes":       0,
	// TODO parse client-output-buffer-limit
	//"client-output-buffer-limit": 0,
}

// RedisClusterChecker is our implementation of RedisClusterCheck intercace
type RedisClusterChecker struct {
	k8sService  k8s.Services
	redisClient redis.Client
	logger      logr.Logger
}

// NewRedisClusterChecker creates an object of the RedisClusterChecker struct
func NewRedisClusterChecker(k8sService k8s.Services, redisClient redis.Client, logger logr.Logger) *RedisClusterChecker {
	return &RedisClusterChecker{
		k8sService:  k8sService,
		redisClient: redisClient,
		logger:      logger,
	}
}

// CheckRedisConfig check current redis config is same as custom config
func (r *RedisClusterChecker) CheckRedisConfig(redisCluster *redisv1beta1.RedisCluster, addr string, auth *util.AuthConfig) error {
	client := goredis.NewClient(&goredis.Options{
		Addr:     net.JoinHostPort(addr, "6379"),
		Password: auth.Password,
		DB:       0,
	})
	defer client.Close()
	configs, err := r.redisClient.GetAllRedisConfig(client)
	if err != nil {
		return err
	}

	for key, value := range redisCluster.Spec.Config {
		var err error
		if _, ok := parseConfigMap[key]; ok {
			value, err = util.ParseRedisMemConf(value)
			if err != nil {
				r.logger.Error(err, "redis config format err", "key", key, "value", value)
				continue
			}
		}
		if value != configs[key] {
			return fmt.Errorf("%s configs conflict, expect: %s, current: %s", key, value, configs[key])
		}
	}
	return nil
}

// CheckRedisNumber controls that the number of deployed redis is the same than the requested on the spec
func (r *RedisClusterChecker) CheckRedisNumber(rc *redisv1beta1.RedisCluster) error {
	ss, err := r.k8sService.GetStatefulSet(rc.Namespace, util.GetRedisName(rc))
	if err != nil {
		return err
	}
	if rc.Spec.Size != *ss.Spec.Replicas {
		return errors.New("number of redis pods differ from specification")
	}
	if rc.Spec.Size != ss.Status.ReadyReplicas {
		return errors.New("waiting all of redis pods become ready")
	}
	return nil
}

// CheckSentinelNumber controls that the number of deployed sentinel is the same than the requested on the spec
func (r *RedisClusterChecker) CheckSentinelNumber(rc *redisv1beta1.RedisCluster) error {
	d, err := r.k8sService.GetStatefulSet(rc.Namespace, util.GetSentinelName(rc))
	if err != nil {
		return err
	}
	if rc.Spec.Sentinel.Replicas != *d.Spec.Replicas {
		return errors.New("number of sentinel pods differ from specification")
	}
	return nil
}

// CheckSentinelReadyReplicas controls that the number of deployed sentinel ready pod is the same than the requested on the spec
func (r *RedisClusterChecker) CheckSentinelReadyReplicas(rc *redisv1beta1.RedisCluster) error {
	d, err := r.k8sService.GetStatefulSet(rc.Namespace, util.GetSentinelName(rc))
	if err != nil {
		return err
	}
	if rc.Spec.Sentinel.Replicas != d.Status.ReadyReplicas {
		return errors.New("waiting all of sentinel pods become ready")
	}
	return nil
}

// CheckAllSlavesFromMaster controls that all slaves have the same master (the real one)
func (r *RedisClusterChecker) CheckAllSlavesFromMaster(master string, rc *redisv1beta1.RedisCluster, auth *util.AuthConfig) error {
	rips, err := r.GetRedisesIPs(rc, auth)
	if err != nil {
		return err
	}
	for _, rip := range rips {
		slave, err := r.redisClient.GetSlaveMasterIP(rip, auth)
		if err != nil {
			return err
		}
		if slave != "" && slave != master {
			return fmt.Errorf("slave %s don't have the master %s, has %s", rip, master, slave)
		}
	}
	return nil
}

// CheckSentinelNumberInMemory controls that sentinels have only the living sentinels on its memory.
func (r *RedisClusterChecker) CheckSentinelNumberInMemory(sentinel string, rc *redisv1beta1.RedisCluster, auth *util.AuthConfig) error {
	nSentinels, err := r.redisClient.GetNumberSentinelsInMemory(sentinel, auth)
	if err != nil {
		return err
	} else if nSentinels != rc.Spec.Sentinel.Replicas {
		return errors.New("sentinels in memory mismatch")
	}
	return nil
}

// CheckSentinelSlavesNumberInMemory controls that sentinels have only the spected slaves number.
func (r *RedisClusterChecker) CheckSentinelSlavesNumberInMemory(sentinel string, rc *redisv1beta1.RedisCluster, auth *util.AuthConfig) error {
	nSlaves, err := r.redisClient.GetNumberSentinelSlavesInMemory(sentinel, auth)
	if err != nil {
		return err
	} else if nSlaves != rc.Spec.Size-1 {
		return errors.New("sentinel's slaves in memory mismatch")
	}
	return nil
}

// CheckSentinelMonitor controls if the sentinels are monitoring the expected master
func (r *RedisClusterChecker) CheckSentinelMonitor(sentinel string, monitor string, auth *util.AuthConfig) error {
	actualMonitorIP, err := r.redisClient.GetSentinelMonitor(sentinel, auth)
	if err != nil {
		return err
	}
	if actualMonitorIP != monitor {
		return errors.New("the monitor on the sentinel config does not match with the expected one")
	}
	return nil
}

// GetMasterIP connects to all redis and returns the master of the redis cluster
func (r *RedisClusterChecker) GetMasterIP(rc *redisv1beta1.RedisCluster, auth *util.AuthConfig) (string, error) {
	rips, err := r.GetRedisesIPs(rc, auth)
	if err != nil {
		return "", err
	}
	masters := []string{}
	for _, rip := range rips {
		master, err := r.redisClient.IsMaster(rip, auth)
		if err != nil {
			return "", err
		}
		if master {
			masters = append(masters, rip)
		}
	}

	if len(masters) != 1 {
		return "", errors.New("number of redis nodes known as master is different than 1")
	}
	return masters[0], nil
}

// GetNumberMasters returns the number of redis nodes that are working as a master
func (r *RedisClusterChecker) GetNumberMasters(rc *redisv1beta1.RedisCluster, auth *util.AuthConfig) (int, error) {
	nMasters := 0
	rips, err := r.GetRedisesIPs(rc, auth)
	if err != nil {
		return nMasters, err
	}
	for _, rip := range rips {
		master, err := r.redisClient.IsMaster(rip, auth)
		if err != nil {
			return nMasters, err
		}
		if master {
			nMasters++
		}
	}
	return nMasters, nil
}

// GetRedisesIPs returns the IPs of the Redis nodes
func (r *RedisClusterChecker) GetRedisesIPs(rc *redisv1beta1.RedisCluster, auth *util.AuthConfig) ([]string, error) {
	redises := []string{}
	rps, err := r.k8sService.GetStatefulSetPods(rc.Namespace, util.GetRedisName(rc))
	if err != nil {
		return nil, err
	}
	for _, rp := range rps.Items {
		if rp.Status.Phase == corev1.PodRunning { // Only work with running pods
			redises = append(redises, rp.Status.PodIP)
		}
	}
	return redises, nil
}

// GetSentinelsIPs returns the IPs of the Sentinel nodes
func (r *RedisClusterChecker) GetSentinelsIPs(rc *redisv1beta1.RedisCluster) ([]string, error) {
	sentinels := []string{}
	rps, err := r.k8sService.GetStatefulSetPods(rc.Namespace, util.GetSentinelName(rc))
	if err != nil {
		return nil, err
	}
	for _, sp := range rps.Items {
		if sp.Status.Phase == corev1.PodRunning { // Only work with running pods
			sentinels = append(sentinels, sp.Status.PodIP)
		}
	}
	return sentinels, nil
}

// GetMinimumRedisPodTime returns the minimum time a pod is alive
func (r *RedisClusterChecker) GetMinimumRedisPodTime(rc *redisv1beta1.RedisCluster) (time.Duration, error) {
	minTime := 100000 * time.Hour // More than ten years
	rps, err := r.k8sService.GetStatefulSetPods(rc.Namespace, util.GetRedisName(rc))
	if err != nil {
		return minTime, err
	}
	for _, redisNode := range rps.Items {
		if redisNode.Status.StartTime == nil {
			continue
		}
		start := redisNode.Status.StartTime.Round(time.Second)
		alive := time.Now().Sub(start)
		r.logger.V(2).Info(fmt.Sprintf("pod %s has been alive for %.f seconds", redisNode.Status.PodIP, alive.Seconds()))
		if alive < minTime {
			minTime = alive
		}
	}
	return minTime, nil
}
