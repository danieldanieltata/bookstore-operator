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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	bookstoreexamplecomv1 "github.com/danieldanieltata/bookstore-operator/api/v1"
)

// BookReconciler reconciles a Book object
type BookReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=bookstore.example.com,resources=books,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bookstore.example.com,resources=books/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=bookstore.example.com,resources=books/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Book object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.1/pkg/reconcile
func (r *BookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	log.Info("Reconciling Book", "request", req)

	book := &bookstoreexamplecomv1.Book{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, book)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Book not found, skipping reconciliation")
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Book found", "book", book.Name, "namespace", book.Namespace)

	var allBooks bookstoreexamplecomv1.BookList
	if err := r.List(ctx, &allBooks); err != nil {
		return ctrl.Result{}, err
	}

	if book.Spec.CopyOf == nil {
		log.Info("Book is original book, counting copies")

		copyCount := 0
		for _, otherBook := range allBooks.Items {
			if otherBook.Spec.CopyOf == nil {
				continue
			}
			if otherBook.Spec.CopyOf.Namespace == book.Namespace && otherBook.Spec.CopyOf.Name == book.Name {
				copyCount++
			}
		}

		log.Info("Counting copies for original book", "book", book.Name, "copyCount", copyCount)

		if book.Status.CopyCount != copyCount {
			book.Status.CopyCount = copyCount
			err = r.Status().Update(ctx, book)
			if err != nil {
				return ctrl.Result{}, err
			}

			log.Info("CopyCount updated", "book", book.Name, "copyCount", copyCount)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bookstoreexamplecomv1.Book{}).
		Watches(
			&bookstoreexamplecomv1.Book{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				other := obj.(*bookstoreexamplecomv1.Book)
				if other.Spec.CopyOf == nil {
					return nil
				}
				if other.Spec.CopyOf.Name == "" || other.Spec.CopyOf.Namespace == "" {
					return nil
				}
				return []reconcile.Request{{
					NamespacedName: types.NamespacedName{
						Namespace: other.Spec.CopyOf.Namespace,
						Name:      other.Spec.CopyOf.Name,
					},
				}}
			}),
		).
		Named("book").
		Complete(r)
}
