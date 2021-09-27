package operator

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	InjectionAnnotation = "config.lunar.tech/cluster-identity-inject"

	// These variables are used for the strategy using KubeControllerManager
	// It is used to find the clusterName in AWS clusters provisioned by Kops
	kubeControllerManagerNamespace               = "kube-system"
	kubeControllerManagerContainerName           = "kube-controller-manager"
	kubeControllerManagerContainerArgumentPrefix = "--cluster-name="

	// These variables are used for the strategy using CoreDNSAutoscaler
	// It is used to find the clusterName in Azure AKS clusters
	CoreDNSAutoScalerLabelKey   = "k8s-app"
	CoreDNSAutoScalerLabelValue = "coredns-autoscaler"
	CoreDNSAutoScalerNamespace  = "kube-system"
	CoreDNSAutoScalerDelimiter  = "-"
)

func GetClusterName(ctx context.Context, apiClient client.Client) (string, error) {
	// We need to try on strategy, and keep going to the next if the first failed

	clusterName, err := KubeControllerGetClusterName(ctx, apiClient)
	if err != nil {
		return "", err
	}
	if clusterName == "" {
		clusterName, err = CoreDNSAutoscalerGetClusterName(ctx, apiClient)
	}
	if err != nil {
		return "", err
	}
	if clusterName == "" {
		return "", fmt.Errorf("could not detect cluster name")
	}
	return clusterName, nil
}

func CoreDNSAutoscalerGetClusterName(ctx context.Context, apiClient client.Client) (string, error) {
	pod, err := getCoreDNSAutoscalerPod(ctx, apiClient)
	if err != nil {
		if err.Error() == "no autoscaler pod found" {
			return "", nil
		}
		return "", err
	}

	return CoreDNSAutoscalerClusterNameFromPod(pod, ctx), nil
}

func KubeControllerGetClusterName(ctx context.Context, apiClient client.Client) (string, error) {
	pod, err := getKubeControllerManagerPod(ctx, apiClient)
	if err != nil {
		if err.Error() == fmt.Sprintf("no %s found", kubeControllerManagerContainerName) {
			return "", nil
		}
		return "", err
	}

	return KubeControllerClusterNameFromPod(&pod), nil
}

func getCoreDNSAutoscalerPod(ctx context.Context, apiClient client.Client) (corev1.Pod, error) {
	var podList corev1.PodList
	err := apiClient.List(ctx, &podList, client.MatchingLabels{CoreDNSAutoScalerLabelKey: CoreDNSAutoScalerLabelValue}, client.InNamespace(CoreDNSAutoScalerNamespace))
	if err != nil {
		return corev1.Pod{}, fmt.Errorf("no CoreDNSAutoscaler found")
	}

	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, "coredns-autoscaler") {
			return pod, nil
		}
	}

	return corev1.Pod{}, fmt.Errorf("no autoscaler pod found")
}

func getKubeControllerManagerPod(ctx context.Context, apiClient client.Client) (corev1.Pod, error) {
	var podList corev1.PodList
	err := apiClient.List(ctx, &podList, client.InNamespace(kubeControllerManagerNamespace))
	if err != nil {
		return corev1.Pod{}, err
	}

	for _, pod := range podList.Items {
		if IsKubeControllerPod(pod.Name) {
			return pod, nil
		}
	}

	return corev1.Pod{}, fmt.Errorf("no %s found", kubeControllerManagerContainerName)
}

func IsKubeControllerPod(podName string) bool {
	return strings.HasPrefix(podName, "kube-controller-manager")
}

// KubeControllerClusterNameFromPod extracts the cluster name from a kube-controller-manager
// Pod definition.
func KubeControllerClusterNameFromPod(pod *corev1.Pod) string {
	for _, container := range pod.Spec.Containers {
		if container.Name != kubeControllerManagerContainerName {
			continue
		}

		name := find(kubeControllerManagerContainerArgumentPrefix, container.Args)
		if name == "" {
			return ""
		}
		return strings.TrimPrefix(name, kubeControllerManagerContainerArgumentPrefix)
	}
	return ""
}

func find(needle string, stack []string) string {
	for _, s := range stack {
		if strings.HasPrefix(s, needle) {
			return s
		}
	}
	return ""
}

// CoreDNSAutoscalerClusterNameFromPod extracts the cluster name from a CoreDNSAutoscaler
// Pod definition.
func CoreDNSAutoscalerClusterNameFromPod(pod corev1.Pod, ctx context.Context) string {
	for _, c := range pod.Spec.Containers {
		for _, e := range c.Env {
			if e.Name == "KUBERNETES_PORT_443_TCP_ADDR" {
				values := strings.Split(e.Value, "-")
				if len(values) > 0 {
					return strings.Split(e.Value, "-")[0]
				}
			}
		}
	}
	return ""
}

func IsNamespaceInjectable(namespace corev1.Namespace) bool {
	return namespace.Annotations[InjectionAnnotation] == "true"
}

func CreateOrUpdateConfigMap(ctx context.Context, apiClient client.Client, nn types.NamespacedName, clusterName string) error {
	var cm corev1.ConfigMap
	err := apiClient.Get(ctx, nn, &cm)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("get ConfigMap '%s': %w", nn, err)
		}

		err := createConfigMap(ctx, apiClient, nn, clusterName)
		if err != nil {
			return fmt.Errorf("create configmap: %w", err)
		}
		return nil
	}

	err = updateConfigMap(ctx, apiClient, cm, clusterName)
	if err != nil {
		return fmt.Errorf("update configmap: %w", err)
	}

	return nil
}

func createConfigMap(ctx context.Context, apiClient client.Client, nn types.NamespacedName, clusterName string) error {
	log.FromContext(ctx).Info(fmt.Sprintf("Creating ConfigMap '%s' with clusterName '%s'", nn.String(), clusterName))

	return apiClient.Create(ctx, &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "cluster-identity",
			Namespace:   nn.Namespace,
			Labels:      nil,
			Annotations: nil,
		},
		Data: map[string]string{
			"clusterName": clusterName,
		},
	})
}

func updateConfigMap(ctx context.Context, apiClient client.Client, cm corev1.ConfigMap, clusterName string) error {
	log.FromContext(ctx).Info(fmt.Sprintf("Updating ConfigMap '%s/%s' with clusterName '%s'", cm.ObjectMeta.Namespace, cm.ObjectMeta.Name, clusterName))
	cm.Data["clusterName"] = clusterName

	return apiClient.Update(ctx, &cm)
}
