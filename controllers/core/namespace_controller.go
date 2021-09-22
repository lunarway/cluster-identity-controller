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

	"github.com/lunarway/cluster-identity-controller/internal/operator"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	ConfigMapKey string
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling namespace")

	var namespace corev1.Namespace
	err := r.Client.Get(ctx, req.NamespacedName, &namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{Requeue: false}, nil
		}
		return ctrl.Result{}, err
	}

	isInjectable := operator.IsNamespaceInjectable(namespace)
	if !isInjectable {
		logger.Info("namespace is not injectable. Skipping.")
		return ctrl.Result{}, nil
	}

	clusterName, err := operator.GetClusterName(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = operator.CreateOrUpdateConfigMap(ctx, r.Client, types.NamespacedName{
		Namespace: req.Name,
		Name:      r.ConfigMapKey,
	}, clusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("store cluster clusterName '%s' in configmap: %w", clusterName, err)
	}

	logger.Info("Completed reconciliation of namespace")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}
