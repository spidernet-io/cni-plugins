package common

import (
	"encoding/json"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"net"
	"strings"
)

type spiderpoolAnnotation struct {
	IPV4 string `json:"ipv4"`
	IPV6 string `json:"ipv6"`
}

func GenerateDeploymentYaml(dpmName, namespace string, labels, annotations map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      dpmName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					//Affinity: &corev1.Affinity{
					//	PodAntiAffinity: &corev1.PodAntiAffinity{
					//		RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					//			{
					//				LabelSelector: &metav1.LabelSelector{
					//					MatchLabels: map[string]string{
					//						"app": dpmName,
					//					},
					//				},
					//				TopologyKey: "beta.kubernetes.io/os",
					//			},
					//		},
					//	},
					//},
					Containers: []corev1.Container{
						{
							Name:            "dao-2048",
							Image:           "ghcr.m.daocloud.io/daocloud/dao-2048:v1.2.0",
							ImagePullPolicy: "IfNotPresent",
							Env: []corev1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "spec.nodeName",
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe: &corev1.Probe{
								FailureThreshold: 3,
								PeriodSeconds:    10,
								SuccessThreshold: 1,
								TimeoutSeconds:   1,
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.IntOrString{
											Type:   intstr.String,
											StrVal: "http",
										},
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							LivenessProbe: &corev1.Probe{
								FailureThreshold: 3,
								PeriodSeconds:    10,
								SuccessThreshold: 1,
								TimeoutSeconds:   1,
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.IntOrString{
											Type:   intstr.String,
											StrVal: "http",
										},
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func GenerateServiceYaml(name, namespace string, port int32, labels map[string]string) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeNodePort,
			IPFamilyPolicy: &ipFamilyPolicy,
			Ports: []corev1.ServicePort{
				{
					Port: port,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: port,
					},
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: labels,
		},
	}
}

func GetAllIPsFromPods(pods *corev1.PodList) ([]string, error) {
	var ips []string
	var err error
	for _, pod := range pods.Items {
		calicoIPs := pod.Annotations["cni.projectcalico.org/podIPs"]
		for _, v := range strings.Split(calicoIPs, ",") {
			netip := net.ParseIP(v)
			if netip.To4() != nil && IPV4 {
				ips = append(ips, netip.String())
			}
			if netip.To4() == nil && IPV6 {
				// TODO: calico ipv6 issue
				// ips = append(ips, netip.String())
			}
		}
		for _, v := range SpiderPoolIPAnnotationsKey {
			spiderpoolIPs, ok := pod.Annotations[v]
			if ok {
				spiderpool := &spiderpoolAnnotation{}
				err = json.Unmarshal([]byte(spiderpoolIPs), spiderpool)
				if err != nil {
					return ips, fmt.Errorf("unmarshal spiderpool annations err: %s", err)
				}
				if spiderpool.IPV4 != "" && IPV4 {
					ip, _, _ := net.ParseCIDR(spiderpool.IPV4)
					ips = append(ips, ip.String())
				}
				if spiderpool.IPV6 != "" && IPV6 {
					ip, _, _ := net.ParseCIDR(spiderpool.IPV6)
					ips = append(ips, ip.String())
				}
			}
		}
	}
	return ips, nil
}

func GetCurlCommandByIPFamily(netIp string, port int32) string {
	args := fmt.Sprintf("%s:%d ", netIp, port)
	if net.ParseIP(netIp).To4() == nil {
		args = fmt.Sprintf("-g [%s]:%d ", netIp, port)
	}
	return "curl " + args
}

func GetPingCommandByIPFamily(netIp string) string {
	command := "ping"
	if net.ParseIP(netIp).To4() == nil {
		command = "ping6"
	}
	return fmt.Sprintf("%s %s -c 3", command, netIp)
}
