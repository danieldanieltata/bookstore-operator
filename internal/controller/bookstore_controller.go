/*
Copyright 2026.

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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	bookstoreexamplecomv1 "github.com/danieldanieltata/bookstore-operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BookStoreReconciler reconciles a BookStore object
type BookStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=bookstore.example.com,resources=bookstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bookstore.example.com,resources=bookstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=bookstore.example.com,resources=bookstores/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BookStore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.1/pkg/reconcile
func (r *BookStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	log.Info("Reconciling BookStore", "request", req)

	// Create namespace with the name of the BookStore if not exists
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
		},
	}

	foundNamespace := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name}, foundNamespace)
	if err != nil && errors.IsNotFound(err) {
		err = r.Create(ctx, namespace)

		if err != nil {
			return ctrl.Result{}, err
		}

		log.Info("Namespace created", "namespace", namespace.Name)
	} else if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BookStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bookstoreexamplecomv1.BookStore{}).
		Named("bookstore").
		Complete(r)
}
