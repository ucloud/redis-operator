package clustercache

import (
	"fmt"
	"sync"

	redisv1beta1 "github.com/ucloud/redis-operator/pkg/apis/redis/v1beta1"

	"github.com/ucloud/redis-operator/pkg/util"
)

type StateType string

const (
	Create StateType = "create"
	Update StateType = "update"
	Check  StateType = "check"
)

// Meta contains RedisCluster some metadata
type Meta struct {
	NameSpace string
	Name      string
	State     StateType
	Size      int32
	Auth      *util.AuthConfig
	Obj       *redisv1beta1.RedisCluster

	Status  redisv1beta1.ConditionType
	Message string

	Config map[string]string
}

func newCluster(rc *redisv1beta1.RedisCluster) *Meta {
	return &Meta{
		Auth: &util.AuthConfig{
			Password: rc.Spec.Password,
		},
		Status:    redisv1beta1.ClusterConditionCreating,
		Config:    rc.Spec.Config,
		Obj:       rc,
		Size:      rc.Spec.Size,
		State:     Create,
		Name:      rc.GetName(),
		NameSpace: rc.GetNamespace(),
		Message:   "Bootstrap redis cluster",
	}
}

// MetaMap cache last RedisCluster and meta data
type MetaMap struct {
	sync.Map
}

func (c *MetaMap) Cache(obj *redisv1beta1.RedisCluster) *Meta {
	meta, ok := c.Load(getNamespacedName(obj.GetNamespace(), obj.GetName()))
	if !ok {
		c.Add(obj)
	} else {
		c.Update(meta.(*Meta), obj)
	}
	return c.Get(obj)
}

func (c *MetaMap) Get(obj *redisv1beta1.RedisCluster) *Meta {
	meta, _ := c.Load(getNamespacedName(obj.GetNamespace(), obj.GetName()))
	return meta.(*Meta)
}

func (c *MetaMap) Add(obj *redisv1beta1.RedisCluster) {
	c.Store(getNamespacedName(obj.GetNamespace(), obj.GetName()), newCluster(obj))
}

func (c *MetaMap) Del(obj *redisv1beta1.RedisCluster) {
	c.Delete(getNamespacedName(obj.GetNamespace(), obj.GetName()))
}

func (c *MetaMap) Update(meta *Meta, new *redisv1beta1.RedisCluster) {
	if meta.Obj.GetGeneration() == new.GetGeneration() {
		meta.State = Check
		return
	}

	old := meta.Obj
	meta.State = Update
	meta.Size = old.Spec.Size
	// Password change is not allowed
	new.Spec.Password = old.Spec.Password
	meta.Auth.Password = old.Spec.Password
	meta.Obj = new

	meta.Status = redisv1beta1.ClusterConditionUpdating
	meta.Message = "Updating redis config"
	if isImagesChanged(old, new) {
		meta.Status = redisv1beta1.ClusterConditionUpgrading
		meta.Message = fmt.Sprintf("Upgrading to %s", new.Spec.Image)
	}
	if isScalingDown(old, new) {
		meta.Status = redisv1beta1.ClusterConditionScalingDown
		meta.Message = fmt.Sprintf("Scaling down form: %d to: %d", meta.Size, new.Spec.Size)
	}
	if isScalingUp(old, new) {
		meta.Status = redisv1beta1.ClusterConditionScaling
		meta.Message = fmt.Sprintf("Scaling up form: %d to: %d", meta.Size, new.Spec.Size)
	}
	if isResourcesChange(old, new) {
		meta.Message = "Updating compute resources"
	}
}

func isImagesChanged(old, new *redisv1beta1.RedisCluster) bool {
	return old.Spec.Image == new.Spec.Image
}

func isScalingDown(old, new *redisv1beta1.RedisCluster) bool {
	return old.Spec.Size > new.Spec.Size
}

func isScalingUp(old, new *redisv1beta1.RedisCluster) bool {
	return old.Spec.Size < new.Spec.Size
}

func isResourcesChange(old, new *redisv1beta1.RedisCluster) bool {
	return old.Spec.Resources.Limits.Memory().Size() != new.Spec.Resources.Limits.Memory().Size() ||
		old.Spec.Resources.Limits.Cpu().Size() != new.Spec.Resources.Limits.Cpu().Size()
}

func getNamespacedName(nameSpace, name string) string {
	return fmt.Sprintf("%s%c%s", nameSpace, '/', name)
}
