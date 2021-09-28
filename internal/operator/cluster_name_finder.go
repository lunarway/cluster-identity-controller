package operator

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type clusterNameStrategy interface {
	GetClusterName(ctx context.Context, apiClient client.Client) (string, error)
}

type ClusterNameFinder struct {
	strategies []clusterNameStrategy
}

func (c *ClusterNameFinder) GetClusterName(ctx context.Context, apiClient client.Client) (string, error) {
	for _, strategy := range c.strategies {
		clusterName, err := strategy.GetClusterName(ctx, apiClient)
		if err != nil {
			return "", err
		}

		if clusterName == "" {
			continue
		}

		return clusterName, nil
	}

	return "", fmt.Errorf("could not detect cluster name")
}

func NewClusterNameFinder() *ClusterNameFinder {
	return &ClusterNameFinder{
		strategies: []clusterNameStrategy{
			&kubeControllerStrategy{},
			&coreDNSClusterNameStrategy{},
		},
	}
}
