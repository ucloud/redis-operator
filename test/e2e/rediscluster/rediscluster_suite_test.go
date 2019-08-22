package rediscluster_test

import (
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/ucloud/redis-operator/test/e2e"
)

var f *e2e.Framework

func TestRediscluster(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Rediscluster Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	f = e2e.NewFramework("test")
	f.BeforeEach()
})

var _ = ginkgo.AfterSuite(func() {
	if f != nil {
		f.AfterEach()
	}
})
