package core

import (
	"context"
	"os"
	"testing"

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

func setupPodReconciler(t *testing.T, configMapKey string, objects []client.Object) (*PodReconciler, client.Client) {
	t.Helper()

	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Namespace{})

	client := fake.NewClientBuilder().
		WithObjects(objects...).
		Build()

	reconciler := &PodReconciler{
		Client:       client,
		ConfigMapKey: configMapKey,
	}

	return reconciler, client
}

func TestPodController(t *testing.T) {
	logf.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))
	var (
		clusterName  = "k8s-202109170606.lunar.tech"
		configMapKey = "cluster-identity"

		controllerManagerPod     = kubeControllerManagerPod(clusterName)
		injectableNamespace      = injectableNamespace()
		nonInjectableNamespace   = nonInjectableNamespace()
		clusterIdentityConfigMap = clusterIdentityConfigMap(injectableNamespace.Name, configMapKey)
	)

	t.Run("inject to injectable namespaces", func(t *testing.T) {
		reconciler, client := setupPodReconciler(t, configMapKey, []client.Object{
			&injectableNamespace,
			&nonInjectableNamespace,
			&controllerManagerPod,
		})

		result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: controllerManagerPod.Namespace,
				Name:      controllerManagerPod.Name,
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)

		checkNamespacesForConfigMap(t, client, injectableNamespace.Name, configMapKey, map[string]string{
			"clusterName": clusterName,
		})
	})

	t.Run("update injectable namespaces", func(t *testing.T) {
		reconciler, client := setupPodReconciler(t, configMapKey, []client.Object{
			&injectableNamespace,
			&nonInjectableNamespace,
			&controllerManagerPod,
			&clusterIdentityConfigMap,
		})

		result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: controllerManagerPod.Namespace,
				Name:      controllerManagerPod.Name,
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
		controllerManagerPod = kubeControllerManagerPod(clusterName)
		controllerManagerPod.Spec.Containers[0].Args = nil

		reconciler, _ := setupPodReconciler(t, configMapKey, []client.Object{
			&injectableNamespace,
			&nonInjectableNamespace,
			&controllerManagerPod,
		})

		result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: controllerManagerPod.Namespace,
				Name:      controllerManagerPod.Name,
			},
		})

		assert.EqualError(t, err, "could not find '--cluster-name=' flag in container 'kube-controller-manager'")
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("not fail when kube-controller-manager is deleted", func(t *testing.T) {
		reconciler, _ := setupPodReconciler(t, configMapKey, []client.Object{
			&injectableNamespace,
			&nonInjectableNamespace,
		})

		result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: controllerManagerPod.Namespace,
				Name:      controllerManagerPod.Name,
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})
}
