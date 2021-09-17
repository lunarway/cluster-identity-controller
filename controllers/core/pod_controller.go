/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/lunarway/cluster-identity-controller/internal/operator"
	"go.uber.org/multierr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	ConfigMapKey string
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling pod")

	if req.Namespace != "kube-system" || !strings.HasPrefix(req.Name, "kube-controller-manager") {
		logger.Info("Not a kube-controller-manager pod. Skipping")
		return ctrl.Result{}, nil
	}

	logger.Info("Reconciling a kube-controller-manager pod")

	var pod corev1.Pod
	err := r.Client.Get(ctx, req.NamespacedName, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Could not find kube-controller-manager pod. Requing")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	clusterName, err := operator.ClusterNameFromPod(&pod)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info(fmt.Sprintf("Found cluster name '%s'", clusterName))

	namespaces, err := r.getNamespacesForInjections(ctx, operator.InjectionAnnotation)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("get namespaces: %w", err)
	}

	logger.Info(fmt.Sprintf("Found %d namespaces: %v", len(namespaces), namespaces))

	err = r.storeInConfigMap(ctx, namespaces, clusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("store cluster name '%s' in configmap: %w", clusterName, err)
	}

	logger.Info("Completed reconciliation of kube-controller-manager-pod")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

func (r *PodReconciler) getNamespacesForInjections(ctx context.Context, injectionSelector string) ([]string, error) {
	var namespaceList corev1.NamespaceList
	err := r.Client.List(ctx, &namespaceList)
	if err != nil {
		return nil, err
	}

	var namespaces []string
	for _, namespace := range namespaceList.Items {
		if namespace.Annotations[injectionSelector] == "true" {
			namespaces = append(namespaces, namespace.Name)
		}
	}
	return namespaces, nil
}

func (r *PodReconciler) storeInConfigMap(ctx context.Context, namespaces []string, clusterName string) error {
	var errs error
	for _, namespace := range namespaces {
		err := r.createOrUpdateConfigMap(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      r.ConfigMapKey,
		}, clusterName)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("get ConfigMap in namespace '%s': %w", namespace, err))
		}
	}

	return errs
}

func (r *PodReconciler) createOrUpdateConfigMap(ctx context.Context, nn types.NamespacedName, clusterName string) error {
	var cm corev1.ConfigMap
	err := r.Client.Get(ctx, nn, &cm)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("get ConfigMap '%s': %w", nn, err)
		}

		err := r.createConfigMap(ctx, nn, clusterName)
		if err != nil {
			return fmt.Errorf("create configmap: %w", err)
		}

		return nil
	}

	err = r.updateConfigMap(ctx, cm, clusterName)
	if err != nil {
		return fmt.Errorf("update configmap: %w", err)
	}

	return nil
}

func (r *PodReconciler) createConfigMap(ctx context.Context, nn types.NamespacedName, clusterName string) error {
	log.FromContext(ctx).Info(fmt.Sprintf("Creating ConfigMap '%s' with clusterName '%s'", nn.String(), clusterName))
	return r.Create(ctx, &corev1.ConfigMap{
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

func (r *PodReconciler) updateConfigMap(ctx context.Context, cm corev1.ConfigMap, clusterName string) error {
	log.FromContext(ctx).Info(fmt.Sprintf("Updating ConfigMap '%s/%s' with clusterName '%s'", cm.ObjectMeta.Namespace, cm.ObjectMeta.Name, clusterName))
	cm.Data["clusterName"] = clusterName
	return r.Update(ctx, &cm)
}
