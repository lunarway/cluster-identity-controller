package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkNamespacesForConfigMap(t *testing.T, client client.Client, injectedNamespace, configMapKey string, configMapData map[string]string) {
	t.Helper()

	var namespaceList corev1.NamespaceList
	err := client.List(context.Background(), &namespaceList)
	assert.NoError(t, err)

	var foundInjectedNamespace bool
	for _, namespace := range namespaceList.Items {
		t.Logf("Checking namespace: %s", namespace.Name)

		configMap, hasConfigMap := namespaceHasConfigMap(t, client, namespace.Name, configMapKey, configMapData)

		if namespace.Name != injectedNamespace {
			require.Falsef(t, hasConfigMap, "namespace '%s' contains a config map and shouldn't", namespace.Name)
			continue
		}

		foundInjectedNamespace = true
		require.Truef(t, hasConfigMap, "namespace '%s' is missing a config map", namespace.Name)

		assert.Equalf(t, configMapData, configMap.Data, "namespace '%s' has the configmap but the data is wrong", namespace.Name)
	}

	if !foundInjectedNamespace {
		t.Fatalf("injected namespace not found")
	}
}

func namespaceHasConfigMap(t *testing.T, client client.Client, namespace, configMap string, configMapData map[string]string) (corev1.ConfigMap, bool) {
	t.Helper()

	fetched := &corev1.ConfigMap{}
	err := client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      configMap,
	}, fetched)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return corev1.ConfigMap{}, false
		}
		assert.NoError(t, err)
	}

	return *fetched, true
}
