package operator

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kubeControllerManagerNamespace               = "kube-system"
	kubeControllerManagerContainerName           = "kube-controller-manager"
	kubeControllerManagerContainerArgumentPrefix = "--cluster-name="
)

type kubeControllerStrategy struct{}

func (k *kubeControllerStrategy) GetClusterName(ctx context.Context, apiClient client.Client) (string, error) {
	pod, err := getKubeControllerManagerPod(ctx, apiClient)
	if err != nil {
		if err.Error() == fmt.Sprintf("no %s found", kubeControllerManagerContainerName) {
			return "", nil
		}
		return "", err
	}

	return KubeControllerClusterNameFromPod(&pod), nil
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

// KubeControllerClusterNameFromPod extracts the cluster name from a kube-controller-manager
// Pod definition.
func kubeControllerClusterNameFromPod(pod *corev1.Pod) string {
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
