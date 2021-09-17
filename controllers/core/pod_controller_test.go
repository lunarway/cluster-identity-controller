package core

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/lunarway/cluster-identity-controller/internal/operator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func setup(t *testing.T, configMapKey string, objects []client.Object) (*PodReconciler, client.Client) {
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
		reconciler, client := setup(t, configMapKey, []client.Object{
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
		reconciler, client := setup(t, configMapKey, []client.Object{
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

		reconciler, _ := setup(t, configMapKey, []client.Object{
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

		assert.EqualError(t, err, "could not find --cluster-name flag in container 'kube-controller-manager'")
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("not fail when kube-controller-manager is deleted", func(t *testing.T) {
		reconciler, _ := setup(t, configMapKey, []client.Object{
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

func clusterIdentityConfigMap(namespace, name string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"otherField":  "other",
			"clusterName": "old",
		},
	}
}

func checkNamespacesForConfigMap(t *testing.T, client client.Client, injectedNamespace, configMapKey string, configMapData map[string]string) {
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

func injectableNamespace() corev1.Namespace {
	return corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "injectable",
			Annotations: map[string]string{
				operator.InjectionAnnotation: "true",
			},
		},
	}
}

func nonInjectableNamespace() corev1.Namespace {
	return corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "non-injectable",
			Annotations: map[string]string{},
		},
	}
}

func kubeControllerManagerPod(clusterName string) corev1.Pod {
	return corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-controller-manager-ip-10-11-12-13.eu-west-1.compute.internal",
			Namespace: "kube-system",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "kube-controller-manager",
					Image: "k8s.gcr.io/kube-controller-manager:v1.20.8",
					Command: []string{
						"/usr/local/bin/kube-controller-manager",
					},
					Args: []string{
						"--allocate-node-cidrs=true",
						"--attach-detach-reconcile-sync-period=1m0s",
						"--cloud-config=/etc/kubernetes/cloud.config",
						"--cloud-provider=aws",
						"--cluster-cidr=100.96.0.0/11",
						fmt.Sprintf("--cluster-name=%s", clusterName),
						"--cluster-signing-cert-file=/srv/kubernetes/ca.crt",
						"--cluster-signing-key-file=/srv/kubernetes/ca.key",
						"--configure-cloud-routes=false",
						"--flex-volume-plugin-dir=/var/lib/kubelet/volumeplugins/",
						"--kubeconfig=/var/lib/kube-controller-manager/kubeconfig",
						"--leader-elect=true",
						"--root-ca-file=/srv/kubernetes/ca.crt",
						"--service-account-private-key-file=/srv/kubernetes/service-account.key",
						"--use-service-account-credentials=true",
						"--v=2",
						"--logtostderr=false",
						"--alsologtostderr",
						"--log-file=/var/log/kube-controller-manager.log",
					},
				},
			},
		},
	}
}
