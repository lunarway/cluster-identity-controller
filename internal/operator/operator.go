package operator

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	InjectionAnnotation = "config.lunar.tech/cluster-identity-inject"
)

func IsKubeControllerPod(podName string) bool {
	return strings.HasPrefix(podName, "kube-controller-manager")
}

func find(needle string, stack []string) string {
	for _, s := range stack {
		if strings.HasPrefix(s, needle) {
			return s
		}
	}
	return ""
}

func IsNamespaceInjectable(namespace corev1.Namespace) bool {
	return namespace.Annotations[InjectionAnnotation] == "true"
}

func CreateOrUpdateConfigMap(ctx context.Context, apiClient client.Client, nn types.NamespacedName, clusterName string) error {
	var cm corev1.ConfigMap
	err := apiClient.Get(ctx, nn, &cm)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("get ConfigMap '%s': %w", nn, err)
		}

		err := createConfigMap(ctx, apiClient, nn, clusterName)
		if err != nil {
			return fmt.Errorf("create configmap: %w", err)
		}
		return nil
	}

	err = updateConfigMap(ctx, apiClient, cm, clusterName)
	if err != nil {
		return fmt.Errorf("update configmap: %w", err)
	}

	return nil
}

func createConfigMap(ctx context.Context, apiClient client.Client, nn types.NamespacedName, clusterName string) error {
	log.FromContext(ctx).Info(fmt.Sprintf("Creating ConfigMap '%s' with clusterName '%s'", nn.String(), clusterName))

	return apiClient.Create(ctx, &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "cluster-identity",
			Namespace:   nn.Namespace,
			Labels:      nil,
			Annotations: nil,
		},
		Data: map[string]string{
			"clusterName": clusterName,
		},
	})
}

func updateConfigMap(ctx context.Context, apiClient client.Client, cm corev1.ConfigMap, clusterName string) error {
	log.FromContext(ctx).Info(fmt.Sprintf("Updating ConfigMap '%s/%s' with clusterName '%s'", cm.ObjectMeta.Namespace, cm.ObjectMeta.Name, clusterName))
	cm.Data["clusterName"] = clusterName

	return apiClient.Update(ctx, &cm)
}
