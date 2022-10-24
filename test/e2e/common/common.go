package common

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"net"
)

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

func GetIPsFromPods(pods *corev1.PodList) []string {
	var ips []string
	for _, pod := range pods.Items {
		for _, podIP := range pod.Status.PodIPs {
			ips = append(ips, podIP.IP)
		}
	}
	return ips
}

func GetCurlCommandByIPFamily(netIp string, port int32) string {
	args := fmt.Sprintf("%s:%d ", netIp, port)
	if net.ParseIP(netIp).To4() == nil {
		args = fmt.Sprintf("-g [%s]:%d", netIp, port)
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
