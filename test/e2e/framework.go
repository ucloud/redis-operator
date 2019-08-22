package e2e

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	redisv1beta1 "github.com/ucloud/redis-operator/pkg/apis/redis/v1beta1"
	"github.com/ucloud/redis-operator/pkg/client/k8s"
	"github.com/ucloud/redis-operator/pkg/client/redis"
	client2 "github.com/ucloud/redis-operator/test/client"
)

var log = logf.Log.WithName("e2e_framework")

// Framework is e2e test framework
type Framework struct {
	Client      *kubernetes.Clientset
	UtilClient  client.Client
	RedisClient redis.Client
	K8sService  k8s.Services
	namespace   string
	config      *restclient.Config
}

// NewFramework create a new Framework with name
func NewFramework(name string) *Framework {
	namespace := fmt.Sprintf("redise2e-%s-%s", name, RandString(8))
	return &Framework{
		namespace: namespace,
	}
}

// Logf write log to ginkgo output
func (f *Framework) Logf(format string, a ...interface{}) {
	l := fmt.Sprintf(format, a...)
	Logf("namespace:%s %s", f.Namespace(), l)
}

// BeforeEach runs before each test
func (f *Framework) BeforeEach() {
	config, err := loadConfig()
	if err != nil {
		panic(err)
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(config).NotTo(gomega.BeNil())
	f.config = config

	client, err := kubernetes.NewForConfig(config)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(client).NotTo(gomega.BeNil())

	cli, err := client2.NewK8sClient(config)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(cli).NotTo(gomega.BeNil())

	f.Client = client
	f.UtilClient = cli
	f.RedisClient = redis.New()
	f.K8sService = k8s.New(cli, log)

	err = f.createTestNamespace()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	f.Logf("test namespace created")

	f.Logf("setup rbac in namespace")
	err = f.createRBAC()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

// AfterEach runs after each test
func (f *Framework) AfterEach() {
	f.Logf("clear rbac in namespace")
	err := f.deleteRBAC()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	err = f.deleteTestNamespace()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	f.Logf("test namespace deleted")
}

// Namespace return the test namespace name
func (f *Framework) Namespace() string {
	return f.namespace
}

// WaitJobSuccess wait for a job to success or timeout
func (f *Framework) WaitJobSuccess(jobName string, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout")
		default:
			job, err := f.Client.BatchV1().Jobs(f.Namespace()).Get(jobName, metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			f.Logf("check job status, expecting 1 succeeded, current %d", job.Status.Succeeded)
			if job.Status.Succeeded == 1 {
				return nil
			}
			time.Sleep(time.Second * 5)
		}
	}
}

// WaitPodRunning wait for a status of a pod become running
func (f *Framework) WaitPodRunning(podName string, timeout time.Duration) (*v1.Pod, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return nil, fmt.Errorf("timeout")
		default:
			pod, err := f.Client.CoreV1().Pods(f.Namespace()).Get(podName, metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			f.Logf("check pod status, expecting Running, current %s", pod.Status.Phase)
			if pod.Status.Phase == v1.PodRunning {
				return pod, nil
			}
			time.Sleep(time.Second * 5)
		}
	}
}

// WaitRedisclusterHealthy wait for a status of a RedisCluster become Healthy
func (f *Framework) WaitRedisclusterHealthy(name string, wait, timeout time.Duration) (result *redisv1beta1.RedisCluster, err error) {
	// wait for redis cluster status change
	time.Sleep(wait)
	result = &redisv1beta1.RedisCluster{}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return nil, fmt.Errorf("timeout")
		default:
			err = f.UtilClient.Get(context.TODO(), types.NamespacedName{
				Namespace: f.namespace,
				Name:      name,
			}, result)
			//err = f.RestClient.Get().Namespace(f.namespace).Resource("*").Name(name).Do().Into(result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			if len(result.Status.Conditions) == 0 {
				time.Sleep(time.Second * 5)
				continue
			}
			f.Logf("check redis cluster status, expecting Healthy, current %s", result.Status.Conditions[0].Type)
			if result.Status.Conditions[0].Type == redisv1beta1.ClusterConditionHealthy {
				return
			}
			time.Sleep(time.Second * 5)
		}
	}
}

// CreateRedisCluster creates a RedisCluster in test namespace
func (f *Framework) CreateRedisCluster(spec *redisv1beta1.RedisCluster) *redisv1beta1.RedisCluster {
	f.Logf("creating RedisCluster %s", spec.Name)
	var err error
	result := &redisv1beta1.RedisCluster{}
	err = f.UtilClient.Get(context.TODO(), types.NamespacedName{
		Namespace: f.namespace,
		Name:      spec.Name,
	}, result)
	if errors.IsNotFound(err) {
		err = f.UtilClient.Create(context.TODO(), spec)
		err = f.UtilClient.Get(context.TODO(), types.NamespacedName{
			Namespace: f.namespace,
			Name:      spec.Name,
		}, result)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		return result
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return result
}

// UpdateRedisCluster update a RedisCluster in test namespace
func (f *Framework) UpdateRedisCluster(spec *redisv1beta1.RedisCluster) *redisv1beta1.RedisCluster {
	f.Logf("updating RedisCluster %s", spec.Name)
	err := f.UtilClient.Update(context.TODO(), spec)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return spec
}

// CreateRedisClusterAndWaitHealthy creates a RedisCluster and waiting for it to become Healthy
func (f *Framework) CreateRedisClusterAndWaitHealthy(spec *redisv1beta1.RedisCluster, timeout time.Duration) *redisv1beta1.RedisCluster {
	result := f.CreateRedisCluster(spec)
	updateResult, err := f.WaitRedisclusterHealthy(result.Name, 30*time.Second, timeout)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return updateResult
}

// DeleteRedisCluster deletes a RedisCluster with specified name in test namespace
func (f *Framework) DeleteRedisCluster(name string) {
	f.Logf("deleting RedisCluster %s", name)
	result := &redisv1beta1.RedisCluster{}
	err := f.UtilClient.Get(context.TODO(), types.NamespacedName{
		Namespace: f.namespace,
		Name:      name,
	}, result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	err = f.UtilClient.Delete(context.TODO(), result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

// CreatePod creates a pod in test namespace
func (f *Framework) CreatePod(spec *v1.Pod) *v1.Pod {
	f.Logf("creating pod %s", spec.Name)
	pod, err := f.Client.CoreV1().Pods(f.Namespace()).Create(spec)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return pod
}

// CreatePodAndWaitRunning creates a pod and waiting for it to become Running
func (f *Framework) CreatePodAndWaitRunning(spec *v1.Pod, timeout time.Duration) *v1.Pod {
	pod := f.CreatePod(spec)
	updatedPod, err := f.WaitPodRunning(pod.Name, timeout)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return updatedPod
}

// DeletePod deletes a pod with specified name in test namespace
func (f *Framework) DeletePod(name string) {
	f.Logf("deleting pod %s", name)
	err := f.Client.CoreV1().Pods(f.Namespace()).Delete(name, nil)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func (f *Framework) createTestNamespace() error {
	nsSpec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: f.namespace},
	}
	_, err := f.Client.CoreV1().Namespaces().Create(nsSpec)
	return err
}

func (f *Framework) deleteTestNamespace() error {
	return f.Client.CoreV1().Namespaces().Delete(f.namespace, &metav1.DeleteOptions{})
}

func (f *Framework) createRBAC() error {
	rbName := f.rolebindingName()
	rbSpec := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: rbName, Namespace: f.namespace},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cr-uae-n1",
		},
		Subjects: []rbacv1.Subject{{
			APIGroup:  "rbac.authorization.k8s.io",
			Kind:      "Group",
			Name:      fmt.Sprintf("system:serviceaccounts:%s", f.namespace),
			Namespace: f.namespace,
		}},
	}
	_, err := f.Client.RbacV1().RoleBindings(f.namespace).Create(rbSpec)
	return err
}

func (f *Framework) deleteRBAC() error {
	rbName := f.rolebindingName()
	return f.Client.RbacV1().RoleBindings(f.namespace).Delete(rbName, &metav1.DeleteOptions{})
}

func (f *Framework) rolebindingName() string {
	return fmt.Sprintf("cr-uae-n1~g-%s", f.namespace)
}

func loadConfig() (*restclient.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		return nil, fmt.Errorf("env KUBECONFIG not set")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}
