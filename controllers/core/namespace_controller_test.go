package core

import (
	"context"
	"os"
	"testing"

	"github.com/lunarway/cluster-identity-controller/internal/operator"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func setupNamespaceReconciler(t *testing.T, configMapKey string, objects []client.Object) (*NamespaceReconciler, client.Client) {
	t.Helper()

	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Namespace{})
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.NamespaceList{})
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Pod{})
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.PodList{})

	client := fake.NewClientBuilder().
		WithObjects(objects...).
		Build()

	reconciler := &NamespaceReconciler{
		Client:            client,
		ConfigMapKey:      configMapKey,
		ClusterNameFinder: operator.NewClusterNameFinder(),
	}

	return reconciler, client
}

func TestNamespaceController(t *testing.T) {
	logf.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))
	var (
		clusterName  = "k8s-202109170606.lunar.tech"
		configMapKey = "cluster-identity"

		controllerManagerPod     = kubeControllerManagerPod(clusterName)
		coreDnsAutoScalerPod     = corednsAutoscalerPod(clusterName)
		injectableNamespace      = injectableNamespace()
		nonInjectableNamespace   = nonInjectableNamespace()
		clusterIdentityConfigMap = clusterIdentityConfigMap(injectableNamespace.Name, configMapKey)
	)

	t.Run("update injectable namespaces via CoreDnsAutoScalerPod", func(t *testing.T) {
		reconciler, client := setupNamespaceReconciler(t, configMapKey, []client.Object{
			&injectableNamespace,
			&coreDnsAutoScalerPod,
			&clusterIdentityConfigMap,
		})

		result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: injectableNamespace.Namespace,
				Name:      injectableNamespace.Name,
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)

		checkNamespacesForConfigMap(t, client, injectableNamespace.Name, configMapKey, map[string]string{
			"otherField":  "other",
			"clusterName": "clustername",
		})
	})

	t.Run("Not inject to nonInjectable namespaces", func(t *testing.T) {
		reconciler, _ := setupNamespaceReconciler(t, configMapKey, []client.Object{
			&controllerManagerPod,
			&nonInjectableNamespace,
		})

		result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: nonInjectableNamespace.Namespace,
				Name:      nonInjectableNamespace.Name,
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("inject to injectable namespaces", func(t *testing.T) {
		reconciler, client := setupNamespaceReconciler(t, configMapKey, []client.Object{
			&controllerManagerPod,
			&injectableNamespace,
		})

		result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: injectableNamespace.Namespace,
				Name:      injectableNamespace.Name,
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)

		checkNamespacesForConfigMap(t, client, injectableNamespace.Name, configMapKey, map[string]string{
			"clusterName": clusterName,
		})
	})

	t.Run("update injectable namespaces via controllerManagerPod", func(t *testing.T) {
		reconciler, client := setupNamespaceReconciler(t, configMapKey, []client.Object{
			&injectableNamespace,
			&controllerManagerPod,
			&clusterIdentityConfigMap,
		})

		result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: injectableNamespace.Namespace,
				Name:      injectableNamespace.Name,
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)

		checkNamespacesForConfigMap(t, client, injectableNamespace.Name, configMapKey, map[string]string{
			"otherField":  "other",
			"clusterName": clusterName,
		})
	})

	t.Run("fail if cluster name cannot be detected", func(t *testing.T) {
		reconciler, _ := setupNamespaceReconciler(t, configMapKey, []client.Object{
			&injectableNamespace,
		})

		result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: injectableNamespace.Namespace,
				Name:      injectableNamespace.Name,
			},
		})
		assert.EqualError(t, err, "could not detect cluster name")
		assert.Equal(t, ctrl.Result{}, result)
	})
}
