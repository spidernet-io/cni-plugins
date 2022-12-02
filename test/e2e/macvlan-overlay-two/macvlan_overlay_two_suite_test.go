package macvlan_overlay_two_test

import (
	"context"
	"fmt"
	multus_v1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/spidernet-io/cni-plugins/pkg/schema"
	"github.com/spidernet-io/cni-plugins/test/e2e/common"
	e2e "github.com/spidernet-io/e2eframework/framework"
	"github.com/spidernet-io/e2eframework/tools"
	spiderdoctorV1 "github.com/spidernet-io/spiderdoctor/pkg/k8s/apis/spiderdoctor.spidernet.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMacvlanOverlayTwo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MacvlanOverlayTwo Suite")
}

var frame *e2e.Framework
var deploymentName, name, namespace string
var spiderDoctorAgent *appsv1.DaemonSet
var label, annotations = make(map[string]string), make(map[string]string)
var successRate = float64(1)
var delayMs = int64(10000)
var testIPv6 = false
var (
	task        *spiderdoctorV1.Nethttp
	plan        *spiderdoctorV1.SchedulePlan
	target      *spiderdoctorV1.NethttpTarget
	targetAgent *spiderdoctorV1.TargetAgentSepc
	request     *spiderdoctorV1.NethttpRequest
	condition   *spiderdoctorV1.NetSuccessCondition
	run         = true
)

var _ = BeforeSuite(func() {
	defer GinkgoRecover()
	var e error
	task = new(spiderdoctorV1.Nethttp)
	plan = new(spiderdoctorV1.SchedulePlan)
	target = new(spiderdoctorV1.NethttpTarget)
	targetAgent = new(spiderdoctorV1.TargetAgentSepc)
	request = new(spiderdoctorV1.NethttpRequest)
	condition = new(spiderdoctorV1.NetSuccessCondition)

	frame, e = e2e.NewFramework(GinkgoT(), []func(*runtime.Scheme) error{multus_v1.AddToScheme, schema.SpiderPoolAddToScheme, spiderdoctorV1.AddToScheme})
	Expect(e).NotTo(HaveOccurred())

	name = "two-macvlan-overlay-" + tools.RandomName()
	deploymentName = "two-macvlan-overlay"
	label["app"] = deploymentName
	namespace = "ns" + tools.RandomName()

	// get macvlan-overlay multus crd instance by name
	multusInstance, err := frame.GetMultusInstance(common.MacvlanOverlayVlan0Name, common.MultusNs)
	Expect(err).NotTo(HaveOccurred())
	Expect(multusInstance).NotTo(BeNil())

	multusInstance, err = frame.GetMultusInstance(common.MacvlanOverlayVlan100Name, common.MultusNs)
	Expect(err).NotTo(HaveOccurred())
	Expect(multusInstance).NotTo(BeNil())

	err = frame.CreateNamespace(namespace)
	Expect(err).NotTo(HaveOccurred())

	annotations[common.MultusAddonAnnotation_Key] = fmt.Sprintf("%s/%s,%s/%s", common.MultusNs, common.MacvlanOverlayVlan0Name, common.MultusNs, common.MacvlanOverlayVlan100Name)
	annotations[common.SpiderPoolIPPoolsAnnotationKey] = `[{"interface": "net1", "ipv4": ["default-v4-ippool"], "ipv6": ["default-v6-ippool"]},{"interface": "net2", "ipv4": ["vlan100-v4"], "ipv6": ["vlan100-v6"]}]`

	GinkgoWriter.Printf("update spiderdoctoragent annotation: %v/%v annotation: %v \n", common.SpiderDoctorAgentNs, common.SpiderDoctorAgentDSName, annotations)
	spiderDoctorAgent, err = frame.GetDaemonSet(common.SpiderDoctorAgentDSName, common.SpiderDoctorAgentNs)
	Expect(err).NotTo(HaveOccurred())
	Expect(spiderDoctorAgent).NotTo(BeNil())

	spiderDoctorAgent.Spec.Template.Annotations = annotations
	err = frame.UpdateResource(spiderDoctorAgent)
	Expect(err).NotTo(HaveOccurred())

	ctx, cancel := context.WithTimeout(context.Background(), 2*common.CtxTimeout)
	defer cancel()
	nodeList, err := frame.GetNodeList()
	Expect(err).NotTo(HaveOccurred())
	err = frame.WaitPodListRunning(spiderDoctorAgent.Spec.Selector.MatchLabels, len(nodeList.Items), ctx)
	Expect(err).NotTo(HaveOccurred())

	time.Sleep(10 * time.Second)
})

var _ = AfterSuite(func() {
	err := frame.DeleteResource(task)
	Expect(err).NotTo(HaveOccurred(), "failed to delete spiderdoctor nethttp %v", name)
	err = frame.DeleteDeploymentUntilFinish(deploymentName, namespace, time.Second*60*10)
	Expect(err).NotTo(HaveOccurred(), "failed to delete deployment %v", deploymentName)
	err = frame.DeleteNamespace(namespace)
	Expect(err).NotTo(HaveOccurred(), "failed to delete namespace %v", namespace)
})
