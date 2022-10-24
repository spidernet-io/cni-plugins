package macvlan_standalone_one_test

import (
	"context"
	"fmt"
	multus_v1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/cni-plugins/pkg/schema"
	"github.com/spidernet-io/cni-plugins/test/e2e/common"
	e2e "github.com/spidernet-io/e2eframework/framework"
	"github.com/spidernet-io/e2eframework/tools"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestMacvlanStandaloneOne(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MacvlanStandaloneOne Suite")
}

var frame *e2e.Framework
var name, namespace, multuNs string

var podList *corev1.PodList
var dp *appsv1.Deployment
var labels, annotations = make(map[string]string), make(map[string]string)

var port int32 = 80
var nodePorts []int32
var podIPs, clusterIPs, nodeIPs []string

var retryTimes = 5

var _ = BeforeSuite(func() {
	defer GinkgoRecover()
	var e error
	frame, e = e2e.NewFramework(GinkgoT(), []func(*runtime.Scheme) error{multus_v1.AddToScheme, schema.SpiderPoolAddToScheme})
	Expect(e).NotTo(HaveOccurred())

	// init namespace name and create
	name = "one-macvlan-standalone"
	namespace = "ns" + tools.RandomName()
	multuNs = "kube-system"
	labels["app"] = name

	err := frame.CreateNamespace(namespace)
	Expect(err).NotTo(HaveOccurred(), "failed to create namespace %v", namespace)
	GinkgoWriter.Printf("create namespace %v \n", namespace)

	// get macvlan-standalone multus crd instance by name
	multusInstance, err := frame.GetMultusInstance(common.MacvlanStandaloneVlan100Name, multuNs)
	Expect(err).NotTo(HaveOccurred())
	Expect(multusInstance).NotTo(BeNil())

	annotations[common.MultusDefaultAnnotationKey] = fmt.Sprintf("%s/%s", multuNs, common.MacvlanStandaloneVlan100Name)

	GinkgoWriter.Printf("create deploy: %v/%v \n", namespace, name)
	dp = common.GenerateDeploymentYaml(name, namespace, labels, annotations)
	Expect(frame.CreateDeployment(dp)).NotTo(HaveOccurred())

	ctx, cancel := context.WithTimeout(context.Background(), 2*common.CtxTimeout)
	defer cancel()

	err = frame.WaitPodListRunning(dp.Spec.Selector.MatchLabels, int(*dp.Spec.Replicas), ctx)
	Expect(err).NotTo(HaveOccurred())

	podList, err = frame.GetDeploymentPodList(dp)
	Expect(err).NotTo(HaveOccurred())
	Expect(podList).NotTo(BeNil())

	// create nodePort service
	st := common.GenerateServiceYaml(name, namespace, port, dp.Spec.Selector.MatchLabels)
	err = frame.CreateService(st)
	Expect(err).NotTo(HaveOccurred())

	GinkgoWriter.Printf("succeed to create nodePort service: %s/%s\n", namespace, name)

	// get clusterIPs & nodePorts
	service, err := frame.GetService(name, namespace)
	Expect(err).NotTo(HaveOccurred())
	Expect(service).NotTo(BeNil(), "failed to get service: %s/%s", namespace, name)
	clusterIPs = service.Spec.ClusterIPs
	nodePorts = common.GetServiceNodePorts(service.Spec.Ports)
	GinkgoWriter.Printf("clusterIPs: %v\n", clusterIPs)

	// check service ready by get endpoint
	err = common.WaitEndpointReady(retryTimes, name, namespace, frame)
	Expect(err).NotTo(HaveOccurred())

	// get pod all ip
	podIPs = common.GetIPsFromPods(podList)
	Expect(podIPs).NotTo(BeNil())
	GinkgoWriter.Printf("Get All PodIPs: %v\n", podIPs)

	// time.Sleep( 500* time.Second)
})

var _ = AfterSuite(func() {
	err := frame.DeleteNamespace(namespace)
	Expect(err).NotTo(HaveOccurred(), "failed to delete namespace %v", namespace)

	err = frame.DeleteDeployment(name, namespace)
	Expect(err).NotTo(HaveOccurred(), "failed to delete deployment %v/%v", namespace, name)

	// delete service
	err = frame.DeleteService(name, namespace)
	Expect(err).To(Succeed())
})
