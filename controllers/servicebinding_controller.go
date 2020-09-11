/*
Copyright 2020 The KubePreset Authors

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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appv1alpha1 "github.com/kubepreset/kubepreset/api/v1alpha1"
)

// ServiceBindingReconciler reconciles a ServiceBinding object
type ServiceBindingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=app.kubepreset.dev,resources=servicebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.kubepreset.dev,resources=servicebindings/status,verbs=get;update;patch

func (r *ServiceBindingReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("servicebinding", req.NamespacedName)

	var sb appv1alpha1.ServiceBinding
	if err := r.Get(ctx, req.NamespacedName, &sb); err != nil {
		log.Error(err, "unable to fetch ServiceBinding")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

func (r *ServiceBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.ServiceBinding{}).
		Complete(r)
}
