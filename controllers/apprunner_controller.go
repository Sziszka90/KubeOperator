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

package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	webappv1 "gomodule/api/v1"
)

// AppRunnerReconciler reconciles a AppRunner object
type AppRunnerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webapp.example.com,resources=operators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webapp.example.com,resources=operators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webapp.example.com,resources=operators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Operator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *AppRunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("reconciling operator")

	var apprunner webappv1.AppRunner
	if err := r.Get(ctx, req.NamespacedName, &apprunner); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	deployment, err := r.desiredDeployment(apprunner)
	if err != nil {
		return ctrl.Result{}, err
	}

	svc, err := r.desiredService(apprunner)
	if err != nil {
		return ctrl.Result{}, err
	}

	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("operator-controller")}

	err = r.Patch(ctx, &deployment, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Patch(ctx, &svc, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	apprunner.Status.URL = urlForService(svc, apprunner.Spec.Frontend.ServingPort)

	err = r.Status().Update(ctx, &apprunner)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciled operator")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppRunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.AppRunner{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
