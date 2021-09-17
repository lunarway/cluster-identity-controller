package operator

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const InjectionAnnotation = "config.lunar.tech/cluster-identity-inject"

// ClusterNameFromPod extracts the cluster name from a kube-controller-manager
// Pod definition.
func ClusterNameFromPod(pod *corev1.Pod) (string, error) {
	containerName := "kube-controller-manager"
	for _, container := range pod.Spec.Containers {
		if container.Name != containerName {
			continue
		}
		argumentPrefix := "--cluster-name="
		name := find(container.Args, argumentPrefix)
		if name == "" {
			return "", fmt.Errorf("could not find --cluster-name flag in container '%s'", container.Name)
		}
		return strings.TrimPrefix(name, argumentPrefix), nil
	}
	return "", fmt.Errorf("could not find container with name '%s'", containerName)
}

func find(stack []string, needle string) string {
	for _, s := range stack {
		if strings.HasPrefix(s, needle) {
			return s
		}
	}
	return ""
}
