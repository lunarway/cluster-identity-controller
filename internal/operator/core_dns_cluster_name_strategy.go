package operator

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	coreDNSAutoScalerLabelKey   = "k8s-app"
	coreDNSAutoScalerLabelValue = "coredns-autoscaler"
	coreDNSAutoScalerNamespace  = "kube-system"
)

type coreDNSClusterNameStrategy struct{}

func (c *coreDNSClusterNameStrategy) GetClusterName(ctx context.Context, apiClient client.Client) (string, error) {
	pod, found, err := getCoreDNSAutoscalerPod(ctx, apiClient)
	if err != nil {
		return "", err
	}

	if !found {
		return "", nil
	}

	return coreDNSAutoscalerClusterNameFromPod(pod, ctx), nil
}

func getCoreDNSAutoscalerPod(ctx context.Context, apiClient client.Client) (corev1.Pod, bool, error) {
	var podList corev1.PodList
	err := apiClient.List(ctx, &podList, client.MatchingLabels{coreDNSAutoScalerLabelKey: coreDNSAutoScalerLabelValue}, client.InNamespace(coreDNSAutoScalerNamespace))
	if err != nil {
		return corev1.Pod{}, false, err
	}

	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, "coredns-autoscaler") {
			return pod, true, nil
		}
	}

	return corev1.Pod{}, false, nil
}

// coreDNSAutoscalerClusterNameFromPod extracts the cluster name from a CoreDNSAutoscaler
// Pod definition.
func coreDNSAutoscalerClusterNameFromPod(pod corev1.Pod, ctx context.Context) string {
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
