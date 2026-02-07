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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	bookstoreexamplecomv1 "github.com/danieldanieltata/bookstore-operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const bookStoreFinalizer = "bookstore.example.com/finalizer"

// BookStoreReconciler reconciles a BookStore object
type BookStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=bookstore.example.com,resources=bookstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bookstore.example.com,resources=bookstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=bookstore.example.com,resources=bookstores/finalizers,verbs=update
// +kubebuilder:rbac:groups=bookstore.example.com,resources=books,verbs=list;delete

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

	// Fetch the BookStore, if not found, it was deleted, do not create or update anything.
	bookstore := &bookstoreexamplecomv1.BookStore{}
	err := r.Get(ctx, req.NamespacedName, bookstore)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion: delete all related Books then remove finalizer.
	if bookstore.DeletionTimestamp != nil {
		log.Info("BookStore is being deleted, running finalizer cleanup", "bookstore", req.NamespacedName, "namespace", bookstore.Namespace)
		if controllerutil.ContainsFinalizer(bookstore, bookStoreFinalizer) {
			if err := r.deleteBooksForBookStore(ctx, bookstore); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(bookstore, bookStoreFinalizer)
			if err := r.Update(ctx, bookstore); err != nil {
				log.Error(err, "Failed to remove finalizer", "bookstore", req.NamespacedName)
				return ctrl.Result{}, err
			}
			log.Info("Finalizer removed, BookStore will be deleted", "bookstore", req.NamespacedName)
		}
		return ctrl.Result{}, nil
	}

	// Ensure finalizer is set so we can clean up on delete.
	if !controllerutil.ContainsFinalizer(bookstore, bookStoreFinalizer) {
		controllerutil.AddFinalizer(bookstore, bookStoreFinalizer)
		if err := r.Update(ctx, bookstore); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Create namespace with the name of the BookStore if it does not exist.
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
		},
	}

	foundNamespace := &corev1.Namespace{}
	err = r.Get(ctx, types.NamespacedName{Name: req.Name}, foundNamespace)
	if err != nil && errors.IsNotFound(err) {
		if err = r.Create(ctx, namespace); err != nil {
			return ctrl.Result{}, err
		}

		bookstore.SetOwnerReferences([]metav1.OwnerReference{
			{
				APIVersion: "v1",
				Kind:       "Namespace",
				Name:       namespace.Name,
				UID:        namespace.UID,
			},
		})
		err = r.Update(ctx, bookstore)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Namespace created", "namespace", namespace.Name)
	} else if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BookStoreReconciler) deleteBooksForBookStore(ctx context.Context, bookstore *bookstoreexamplecomv1.BookStore) error {
	log := logf.FromContext(ctx)

	// bookstore namespace is the same as the bookstore name, i wasn't sure if we wanted to put the bookstore in the created namespace or not.
	bookstoreNS := bookstore.Name

	var inNSList bookstoreexamplecomv1.BookList
	if err := r.List(ctx, &inNSList, client.InNamespace(bookstoreNS)); err != nil {
		return err
	}
	for i := range inNSList.Items {
		b := &inNSList.Items[i]
		if err := r.Delete(ctx, b); err != nil && !errors.IsNotFound(err) {
			return err
		}
		log.Info("Deleted Book in bookstore namespace", "book", b.Name, "namespace", b.Namespace)
	}

	var allBooks bookstoreexamplecomv1.BookList
	if err := r.List(ctx, &allBooks); err != nil {
		return err
	}
	for i := range allBooks.Items {
		b := &allBooks.Items[i]

		if b.Spec.CopyOf == nil || b.Spec.CopyOf.Namespace != bookstoreNS {
			continue
		}
		if err := r.Delete(ctx, b); err != nil && !errors.IsNotFound(err) {
			return err
		}
		log.Info("Deleted copy Book", "book", b.Name, "namespace", b.Namespace, "copyOf", b.Spec.CopyOf)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BookStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bookstoreexamplecomv1.BookStore{}).
		Named("bookstore").
		Complete(r)
}
