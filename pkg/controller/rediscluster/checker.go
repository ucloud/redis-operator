package rediscluster

import (
	"errors"
	"fmt"
	"time"

	redisv1beta1 "github.com/ucloud/redis-operator/pkg/apis/redis/v1beta1"
	"github.com/ucloud/redis-operator/pkg/controller/clustercache"
	"github.com/ucloud/redis-operator/pkg/util"
)

const (
	checkInterval  = 5 * time.Second
	timeOut        = 30 * time.Second
	needRequeueMsg = "need requeue"
)

var (
	needRequeueErr = errors.New(needRequeueMsg)
)

// CheckAndHeal Check the health of the cluster and heal,
// Waiting Number of ready redis is equal as the set on the RedisCluster spec
// Waiting Number of ready sentinel is equal as the set on the RedisCluster spec
// Check only one master
// Number of redis master is 1
// All redis slaves have the same master
// Set Custom Redis config
// All sentinels points to the same redis master
// Sentinel has not death nodes
// Sentinel knows the correct slave number
func (r *RedisClusterHandler) CheckAndHeal(meta *clustercache.Meta) error {
	if err := r.rcChecker.CheckRedisNumber(meta.Obj); err != nil {
		r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).V(2).Info("number of redis mismatch, this could be for a change on the statefulset")
		r.eventsCli.UpdateCluster(meta.Obj, "wait for all redis server start")
		return needRequeueErr
	}
	if err := r.rcChecker.CheckSentinelNumber(meta.Obj); err != nil {
		r.eventsCli.FailedCluster(meta.Obj, err.Error())
		return nil
	}

	nMasters, err := r.rcChecker.GetNumberMasters(meta.Obj, meta.Auth)
	if err != nil {
		return err
	}
	switch nMasters {
	case 0:
		r.eventsCli.UpdateCluster(meta.Obj, "set master")
		r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).V(2).Info("no master find, fixing...")
		redisesIP, err := r.rcChecker.GetRedisesIPs(meta.Obj, meta.Auth)
		if err != nil {
			return err
		}
		if len(redisesIP) == 1 {
			if err := r.rcHealer.MakeMaster(redisesIP[0], meta.Auth); err != nil {
				return err
			}
			break
		}
		minTime, err := r.rcChecker.GetMinimumRedisPodTime(meta.Obj)
		if err != nil {
			return err
		}
		r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).Info(fmt.Sprintf("time %.f more than expected. Not even one master, fixing...", minTime.Round(time.Second).Seconds()))
		if err := r.rcHealer.SetOldestAsMaster(meta.Obj, meta.Auth); err != nil {
			return err
		}
	case 1:
		break
	default:
		return errors.New("more than one master, fix manually")
	}

	master, err := r.rcChecker.GetMasterIP(meta.Obj, meta.Auth)
	if err != nil {
		return err
	}
	if err := r.rcChecker.CheckAllSlavesFromMaster(master, meta.Obj, meta.Auth); err != nil {
		r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).Info(err.Error())
		if err := r.rcHealer.SetMasterOnAll(master, meta.Obj, meta.Auth); err != nil {
			return err
		}
	}

	if err = r.setRedisConfig(meta); err != nil {
		return err
	}

	sentinels, err := r.rcChecker.GetSentinelsIPs(meta.Obj)
	if err != nil {
		return err
	}
	for _, sip := range sentinels {
		if err := r.rcChecker.CheckSentinelMonitor(sip, master, meta.Auth); err != nil {
			r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).Info(err.Error())
			if err := r.rcHealer.NewSentinelMonitor(sip, master, meta.Obj, meta.Auth); err != nil {
				return err
			}
		}
	}
	for _, sip := range sentinels {
		if err := r.rcChecker.CheckSentinelSlavesNumberInMemory(sip, meta.Obj, meta.Auth); err != nil {
			r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).
				Info("restoring sentinel ...", "sentinel", sip, "reason", err.Error())
			if err := r.rcHealer.RestoreSentinel(sip, meta.Auth); err != nil {
				return err
			}
			if err := r.waitRestoreSentinelSlavesOK(sip, meta.Obj, meta.Auth); err != nil {
				r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).Info(err.Error())
				return err
			}
		}
	}
	for _, sip := range sentinels {
		if err := r.rcChecker.CheckSentinelNumberInMemory(sip, meta.Obj, meta.Auth); err != nil {
			r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).
				Info("restoring sentinel ...", "sentinel", sip, "reason", err.Error())
			if err := r.rcHealer.RestoreSentinel(sip, meta.Auth); err != nil {
				return err
			}
		}
	}

	if err = r.setSentinelConfig(meta, sentinels); err != nil {
		return err
	}

	return nil
}

func (r *RedisClusterHandler) setRedisConfig(meta *clustercache.Meta) error {
	redises, err := r.rcChecker.GetRedisesIPs(meta.Obj, meta.Auth)
	if err != nil {
		return err
	}
	for _, rip := range redises {
		if err := r.rcChecker.CheckRedisConfig(meta.Obj, rip, meta.Auth); err != nil {
			r.logger.WithValues("namespace", meta.Obj.Namespace, "name", meta.Obj.Name).Info(err.Error())
			r.eventsCli.UpdateCluster(meta.Obj, "set custom config for redis server")
			if err := r.rcHealer.SetRedisCustomConfig(rip, meta.Obj, meta.Auth); err != nil {
				return err
			}
		}
	}
	return nil
}

// TODO do as set redis config
func (r *RedisClusterHandler) setSentinelConfig(meta *clustercache.Meta, sentinels []string) error {
	if meta.State == clustercache.Check {
		return nil
	}

	for _, sip := range sentinels {
		if err := r.rcHealer.SetSentinelCustomConfig(sip, meta.Obj, meta.Auth); err != nil {
			return err
		}
	}
	return nil
}

func (r *RedisClusterHandler) waitRestoreSentinelSlavesOK(sentinel string, rc *redisv1beta1.RedisCluster, auth *util.AuthConfig) error {
	timer := time.NewTimer(timeOut)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("wait for resetore sentinel slave timeout")
		default:
			if err := r.rcChecker.CheckSentinelSlavesNumberInMemory(sentinel, rc, auth); err != nil {
				r.logger.WithValues("namespace", rc.Namespace, "name", rc.Name).Info(err.Error())
				time.Sleep(checkInterval)
			} else {
				return nil
			}
		}
	}
}
