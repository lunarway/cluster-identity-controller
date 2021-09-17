package operator

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	InjectionAnnotation = "config.lunar.tech/cluster-identity-inject"

	kubeControllerManagerContainerName           = "kube-controller-manager"
	kubeControllerManagerContainerArgumentPrefix = "--cluster-name="
)

// ClusterNameFromPod extracts the cluster name from a kube-controller-manager
// Pod definition.
func ClusterNameFromPod(pod *corev1.Pod) (string, error) {
	for _, container := range pod.Spec.Containers {
		if container.Name != kubeControllerManagerContainerName {
			continue
		}

		name := find(container.Args, kubeControllerManagerContainerArgumentPrefix)
		if name == "" {
			return "", fmt.Errorf("could not find '%s' flag in container '%s'", kubeControllerManagerContainerArgumentPrefix, container.Name)
		}
		return strings.TrimPrefix(name, kubeControllerManagerContainerArgumentPrefix), nil
	}
	return "", fmt.Errorf("could not find container with name '%s'", kubeControllerManagerContainerName)
}

func find(stack []string, needle string) string {
	for _, s := range stack {
		if strings.HasPrefix(s, needle) {
			return s
		}
	}
	return ""
}
