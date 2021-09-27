package operator

import (
	"context"
	"fmt"
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
	pod, err := getCoreDNSAutoscalerPod(ctx, apiClient)
	if err != nil {
		if err.Error() == "no autoscaler pod found" {
			return "", nil
		}
		return "", err
	}

	return coreDNSAutoscalerClusterNameFromPod(pod, ctx), nil
}

func getCoreDNSAutoscalerPod(ctx context.Context, apiClient client.Client) (corev1.Pod, error) {
	var podList corev1.PodList
	err := apiClient.List(ctx, &podList, client.MatchingLabels{coreDNSAutoScalerLabelKey: coreDNSAutoScalerLabelValue}, client.InNamespace(coreDNSAutoScalerNamespace))
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
