package operator

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	InjectionAnnotation = "config.lunar.tech/cluster-identity-inject"

	kubeControllerManagerNamespace               = "kube-system"
	kubeControllerManagerContainerName           = "kube-controller-manager"
	kubeControllerManagerContainerArgumentPrefix = "--cluster-name="

	CoreDNSAutoScalerLabelKey   = "k8s-app"
	CoreDNSAutoScalerLabelValue = "coredns-autoscaler"
	CoreDNSAutoScalerNamespace  = "kube-system"

	clusterName = ""
)

func GetClusterName(ctx context.Context, apiClient client.Client) (string, error) {

	//clusterName, err := KubeControllerGetClusterName(ctx, apiClient)
	//if err != nil {
	//	return "", err
	//}
	clusterName, err := CoreDNSAutoscalerGetClusterName(ctx, apiClient)
	if err != nil {
		return "", err
	}

	return clusterName, nil
}

func CoreDNSAutoscalerGetClusterName(ctx context.Context, apiClient client.Client) (string, error) {
	pod, err := getCoreDNSAutoscalerPod(ctx, apiClient)
	if err != nil {
		return "", err
	}

	clusterName, err := CoreDNSAutoscalerClusterNameFromPod(&pod)
	if err != nil {
		return "", err
	}
	return clusterName, nil
}

func KubeControllerGetClusterName(ctx context.Context, apiClient client.Client) (string, error) {
	pod, err := getKubeControllerManagerPod(ctx, apiClient)
	if err != nil {
		return "", err
	}

	clusterName, err := KubeControllerClusterNameFromPod(&pod)
	if err != nil {
		return "", err
	}
	return clusterName, nil
}

func getCoreDNSAutoscalerPod(ctx context.Context, apiClient client.Client) (corev1.Pod, error) {
	var podList corev1.PodList

	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{CoreDNSAutoScalerLabelKey: CoreDNSAutoScalerLabelValue}}

	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		Limit:         1,
	}

	// https://stackoverflow.com/questions/63142663/using-client-go-api-to-list-pods-managed-by-a-deployment-controller-not-working
	// https://pkg.go.dev/k8s.io/client-go@v11.0.0+incompatible/kubernetes/typed/core/v1#PodInterface
	// https://pkg.go.dev/k8s.io/client-go/kubernetes/typed/core/v1#CoreV1Client.Pods
	podList, err := metav1.Pods(CoreDNSAutoScalerNamespace).List(listOptions)

	if err != nil {
		return podList.Items[0], nil
	}

	return corev1.Pod{}, fmt.Errorf("no %s found", kubeControllerManagerContainerName)
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
func KubeControllerClusterNameFromPod(pod *corev1.Pod) (string, error) {
	for _, container := range pod.Spec.Containers {
		if container.Name != kubeControllerManagerContainerName {
			continue
		}

		name := find(kubeControllerManagerContainerArgumentPrefix, container.Args)
		if name == "" {
			return "", fmt.Errorf("could not find '%s' flag in container '%s'", kubeControllerManagerContainerArgumentPrefix, container.Name)
		}
		return strings.TrimPrefix(name, kubeControllerManagerContainerArgumentPrefix), nil
	}
	return "", fmt.Errorf("could not find container with name '%s'", kubeControllerManagerContainerName)
}

func find(needle string, stack []string) string {
	for _, s := range stack {
		if strings.HasPrefix(s, needle) {
			return s
		}
	}
	return ""
}

// ClusterNameFromPod extracts the cluster name from a kube-controller-manager
// Pod definition.
func CoreDNSAutoscalerClusterNameFromPod(pod *corev1.Pod) (string, error) {
	for _, container := range pod.Spec.Containers {
		if container.Name != kubeControllerManagerContainerName {
			continue
		}

		name := find(kubeControllerManagerContainerArgumentPrefix, container.Args)
		if name == "" {
			return "", fmt.Errorf("could not find '%s' flag in container '%s'", kubeControllerManagerContainerArgumentPrefix, container.Name)
		}
		return strings.TrimPrefix(name, kubeControllerManagerContainerArgumentPrefix), nil
	}
	return "", fmt.Errorf("could not find container with name '%s'", kubeControllerManagerContainerName)
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
