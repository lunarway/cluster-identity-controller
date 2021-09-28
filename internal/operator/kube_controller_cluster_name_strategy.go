package operator

import (
	"context"
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
	pod, found, err := getKubeControllerManagerPod(ctx, apiClient)
	if err != nil {
		return "", err
	}

	if !found {
		return "", nil
	}

	return kubeControllerClusterNameFromPod(&pod), nil
}

func getKubeControllerManagerPod(ctx context.Context, apiClient client.Client) (corev1.Pod, bool, error) {
	var podList corev1.PodList
	err := apiClient.List(ctx, &podList, client.InNamespace(kubeControllerManagerNamespace))
	if err != nil {
		return corev1.Pod{}, false, err
	}

	for _, pod := range podList.Items {
		if IsKubeControllerPod(pod.Name) {
			return pod, true, nil
		}
	}

	return corev1.Pod{}, false, nil
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
