package operator

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestClusterNameFinderGetClusterName(t *testing.T) {
	var (
		ctx = context.Background()
	)

	t.Run("Return no cluster when no strategies exist", func(t *testing.T) {
		sut := &ClusterNameFinder{}
		apiClient := fake.NewClientBuilder().Build()

		clusterName, err := sut.GetClusterName(ctx, apiClient)

		assert.Equal(t, "", clusterName)
		assert.Error(t, err)
	})

	t.Run("Return no cluster name when strategy does not find cluster name", func(t *testing.T) {
		sut := &ClusterNameFinder{
			strategies: []clusterNameStrategy{newFakeStrategy("", nil)},
		}
		apiClient := fake.NewClientBuilder().Build()

		clusterName, err := sut.GetClusterName(ctx, apiClient)

		assert.Equal(t, "", clusterName)
		assert.EqualError(t, err, "could not detect cluster name")
	})

	t.Run("Return error when strategy fails", func(t *testing.T) {
		expectedErr := fmt.Errorf("error")
		sut := &ClusterNameFinder{
			strategies: []clusterNameStrategy{newFakeStrategy("", expectedErr)},
		}
		apiClient := fake.NewClientBuilder().Build()

		clusterName, err := sut.GetClusterName(ctx, apiClient)

		assert.Equal(t, "", clusterName)
		assert.EqualError(t, expectedErr, err.Error())
	})

	t.Run("Return cluster name from strategy", func(t *testing.T) {
		expectedClusterName := "clusterName"
		sut := &ClusterNameFinder{
			strategies: []clusterNameStrategy{newFakeStrategy(expectedClusterName, nil)},
		}
		apiClient := fake.NewClientBuilder().Build()

		clusterName, err := sut.GetClusterName(ctx, apiClient)

		assert.Equal(t, expectedClusterName, clusterName)
		assert.NoError(t, err)
	})

	t.Run("Return cluster name from strategy after first strategy fails", func(t *testing.T) {
		expectedClusterName := "clusterName"
		sut := &ClusterNameFinder{
			strategies: []clusterNameStrategy{
				newFakeStrategy("", nil),
				newFakeStrategy(expectedClusterName, nil)},
		}
		apiClient := fake.NewClientBuilder().Build()

		clusterName, err := sut.GetClusterName(ctx, apiClient)

		assert.Equal(t, expectedClusterName, clusterName)
		assert.NoError(t, err)
	})
}

type fakeStrategy struct {
	clusterName string
	err         error
}

func newFakeStrategy(clusterName string, err error) *fakeStrategy {
	return &fakeStrategy{
		clusterName: clusterName,
		err:         err,
	}
}

func (f *fakeStrategy) GetClusterName(context.Context, client.Client) (string, error) {
	if f.err != nil {
		return "", f.err
	}

	return f.clusterName, nil
}
