package operator

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nodeLabel = "clusterName"
)

type nodeLabelStrategy struct{}

func (k *nodeLabelStrategy) GetClusterName(ctx context.Context, apiClient client.Client) (string, error) {

	node, found, err := getNodeWithClusterNameLabel(ctx, apiClient)
	if err != nil {
		return "", err
	}

	if !found {
		return "", nil
	}

	return node.Labels[nodeLabel], nil
}

func getNodeWithClusterNameLabel(ctx context.Context, apiClient client.Client) (corev1.Node, bool, error) {
	var nodeList corev1.NodeList
	err := apiClient.List(ctx, &nodeList, client.HasLabels{nodeLabel})
	if err != nil {
		return corev1.Node{}, false, err
	}

	for _, node := range nodeList.Items {
		if node.Labels[nodeLabel] != "" {
			return node, true, nil
		}
	}

	return corev1.Node{}, false, nil
}
