package clustercache

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1beta1 "github.com/ucloud/redis-operator/pkg/apis/redis/v1beta1"
)

func TestCache(t *testing.T) {
	tests := []struct {
		name string
		rc1  *redisv1beta1.RedisCluster
		rc2  *redisv1beta1.RedisCluster
		rc3  *redisv1beta1.RedisCluster
	}{
		{
			name: "update",
			rc1: &redisv1beta1.RedisCluster{
				ObjectMeta: v1.ObjectMeta{
					Name:       "test1",
					Namespace:  "prj-shu",
					Generation: 1,
				},
				Spec: redisv1beta1.RedisClusterSpec{
					Size: 3,
				},
			},
			rc2: &redisv1beta1.RedisCluster{
				ObjectMeta: v1.ObjectMeta{
					Name:       "test1",
					Namespace:  "prj-shu",
					Generation: 2,
				},
				Spec: redisv1beta1.RedisClusterSpec{
					Size: 4,
				},
			},
			rc3: &redisv1beta1.RedisCluster{
				ObjectMeta: v1.ObjectMeta{
					Name:       "test1",
					Namespace:  "prj-shu",
					Generation: 2,
				},
				Spec: redisv1beta1.RedisClusterSpec{
					Size: 4,
				},
			},
		},
	}

	meta := make(MetaMap)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rcMeta1 := meta.Cache(test.rc1)
			assert.EqualValues(t, redisv1beta1.ClusterConditionCreating, rcMeta1.Status)
			assert.EqualValues(t, Create, rcMeta1.State)
			rcMeta2 := meta.Cache(test.rc2)
			assert.EqualValues(t, redisv1beta1.ClusterConditionScaling, rcMeta2.Status)
			assert.EqualValues(t, Update, rcMeta1.State)
			rcMeta3 := meta.Cache(test.rc3)
			assert.EqualValues(t, redisv1beta1.ClusterConditionScaling, rcMeta3.Status)
			assert.EqualValues(t, Check, rcMeta3.State)
		})
	}
}

func TestCachePasswd(t *testing.T) {
	tests := []struct {
		name string
		rc1  *redisv1beta1.RedisCluster
		rc2  *redisv1beta1.RedisCluster
		rc3  *redisv1beta1.RedisCluster
	}{
		{
			name: "update",
			rc1: &redisv1beta1.RedisCluster{
				ObjectMeta: v1.ObjectMeta{
					Name:       "test1",
					Namespace:  "prj-xxx",
					Generation: 1,
				},
				Spec: redisv1beta1.RedisClusterSpec{
					Size:     3,
					Password: "test",
				},
			},
			rc2: &redisv1beta1.RedisCluster{
				ObjectMeta: v1.ObjectMeta{
					Name:       "test1",
					Namespace:  "prj-xxx",
					Generation: 2,
				},
				Spec: redisv1beta1.RedisClusterSpec{
					Size:     4,
					Password: "test1",
				},
			},
			rc3: &redisv1beta1.RedisCluster{
				ObjectMeta: v1.ObjectMeta{
					Name:       "test1",
					Namespace:  "prj-xxx",
					Generation: 3,
				},
				Spec: redisv1beta1.RedisClusterSpec{
					Size:     4,
					Password: "test2",
				},
			},
		},
	}

	meta := make(MetaMap)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rcMeta1 := meta.Cache(test.rc1)
			assert.EqualValues(t, "test", rcMeta1.Auth.Password)
			assert.EqualValues(t, "test", rcMeta1.Obj.Spec.Password)
			assert.EqualValues(t, "test", test.rc1.Spec.Password)
			rcMeta2 := meta.Cache(test.rc2)
			assert.EqualValues(t, "test", rcMeta2.Auth.Password)
			assert.EqualValues(t, "test", rcMeta2.Obj.Spec.Password)
			assert.EqualValues(t, "test", test.rc2.Spec.Password)
			rcMeta3 := meta.Cache(test.rc3)
			assert.EqualValues(t, "test", rcMeta3.Auth.Password)
			assert.EqualValues(t, "test", rcMeta3.Obj.Spec.Password)
			assert.EqualValues(t, "test", test.rc3.Spec.Password)
		})
	}
}
