package core

import (
	"fmt"

	"github.com/lunarway/cluster-identity-controller/internal/operator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func corednsAutoscalerPod(clusterName string) corev1.Pod {
	return corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coredns-autoscaler-foo-bar",
			Namespace: "kube-system",
			Labels: map[string]string{
				"k8s-app":           "coredns-autoscaler",
				"pod-template-hash": "54d55c8b75",
			},
			Annotations: map[string]string{
				"cluster-autoscaler.kubernetes.io/safe-to-evict": "true",
				"seccomp.security.alpha.kubernetes.io/pod":       "runtime/default",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "autoscaler",
					Image: "mcr.microsoft.com/oss/kubernetes/autoscaler/cluster-proportional-autoscaler:1.8.3",
					Command: []string{
						"/cluster-proportional-autoscaler",
					},
					Args: []string{
						"--namespace=kube-system",
						"--configmap=coredns-autoscaler",
						"--target=deployment/coredns",
						"--default-params={\"ladder\":{\"coresToReplicas\":[[1,2],[512,3],[1024,4],[2048,5]],\"nodesToReplicas\":[[1,2],[8,3],[16,4],[32,5]]}}",
						"--logtostderr=true",
						"--v=2",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "KUBERNETES_PORT_443_TCP_ADDR",
							Value: "clustername-dns-42fc8372.hcp.westeurope.azmk8s.io",
						},
						{
							Name:  "KUBERNETES_PORT",
							Value: "tcp://clustername-dns-42fc8372.hcp.westeurope.azmk8s.io:443",
						},
						{
							Name:  "KUBERNETES_PORT_443_TCP",
							Value: "tcp://clustername-dns-42fc8372.hcp.westeurope.azmk8s.io:443",
						},
						{
							Name:  "KUBERNETES_SERVICE_HOST",
							Value: "clustername-dns-42fc8372.hcp.westeurope.azmk8s.io",
						},
					},
				},
			},
		},
	}
}

func kubeControllerManagerPod(clusterName string) corev1.Pod {
	return corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-controller-manager-ip-10-11-12-13.eu-west-1.compute.internal",
			Namespace: "kube-system",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "kube-controller-manager",
					Image: "registry.k8s.io/kube-controller-manager:v1.20.8",
					Command: []string{
						"/usr/local/bin/kube-controller-manager",
					},
					Args: []string{
						"--allocate-node-cidrs=true",
						"--attach-detach-reconcile-sync-period=1m0s",
						"--cloud-config=/etc/kubernetes/cloud.config",
						"--cloud-provider=aws",
						"--cluster-cidr=100.96.0.0/11",
						fmt.Sprintf("--cluster-name=%s", clusterName),
						"--cluster-signing-cert-file=/srv/kubernetes/ca.crt",
						"--cluster-signing-key-file=/srv/kubernetes/ca.key",
						"--configure-cloud-routes=false",
						"--flex-volume-plugin-dir=/var/lib/kubelet/volumeplugins/",
						"--kubeconfig=/var/lib/kube-controller-manager/kubeconfig",
						"--leader-elect=true",
						"--root-ca-file=/srv/kubernetes/ca.crt",
						"--service-account-private-key-file=/srv/kubernetes/service-account.key",
						"--use-service-account-credentials=true",
						"--v=2",
						"--logtostderr=false",
						"--alsologtostderr",
						"--log-file=/var/log/kube-controller-manager.log",
					},
				},
			},
		},
	}
}

func nodeWithClusterNameLabel(clusterName string) corev1.Node {
	return corev1.Node{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Node",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "gke-node",
			Labels: map[string]string{
				"clusterName": clusterName,
			},
		},
		Spec: corev1.NodeSpec{},
	}
}

func injectableNamespace() corev1.Namespace {
	return corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "injectable",
			Annotations: map[string]string{
				operator.InjectionAnnotation: "true",
			},
		},
	}
}

func nonInjectableNamespace() corev1.Namespace {
	return corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "non-injectable",
			Annotations: map[string]string{},
		},
	}
}

func clusterIdentityConfigMap(namespace, name string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"otherField":  "other",
			"clusterName": "old",
		},
	}
}
