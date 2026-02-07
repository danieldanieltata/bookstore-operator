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

package v1

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	bookstoreexamplecomv1 "github.com/danieldanieltata/bookstore-operator/api/v1"
)

// nolint:unused
// log is for logging in this package.
var booklog = logf.Log.WithName("book-resource")

// SetupBookWebhookWithManager registers the webhook for Book in the manager.
func SetupBookWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &bookstoreexamplecomv1.Book{}).
		WithValidator(&BookCustomValidator{Client: mgr.GetClient(), Reader: mgr.GetAPIReader()}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-bookstore-example-com-v1-book,mutating=false,failurePolicy=fail,sideEffects=None,groups=bookstore.example.com,resources=books,verbs=create;update,versions=v1,name=vbook-v1.kb.io,admissionReviewVersions=v1

// BookCustomValidator struct is responsible for validating the Book resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type BookCustomValidator struct {
	Client client.Client
	Reader client.Reader
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Book.
func (v *BookCustomValidator) ValidateCreate(ctx context.Context, obj *bookstoreexamplecomv1.Book) (admission.Warnings, error) {
	booklog.Info("Validation for Book upon creation", "name", obj.GetName())

	if err := v.validateCopyOfReference(ctx, obj); err != nil {
		return nil, err
	}

	if !hasAtLeastOneOverride(&obj.Spec) {
		return nil, fmt.Errorf("a Book with copyOf must override title, price, or genre in spec")
	}

	if !hasRequiredFieldsWhenNotCopy(&obj.Spec) {
		return nil, fmt.Errorf("a Book without copyOf must have spec.title, spec.price, and spec.genre set (non-zero)")
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Book.
func (v *BookCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *bookstoreexamplecomv1.Book) (admission.Warnings, error) {
	booklog.Info("Validation for Book upon update", "name", newObj.GetName())

	if err := v.validateCopyOfReference(ctx, newObj); err != nil {
		return nil, err
	}

	if !hasAtLeastOneOverride(&newObj.Spec) {
		return nil, fmt.Errorf("when spec.copyOf is set, at least one of spec.title, spec.price, or spec.genre must be set (non-zero)")
	}

	if !hasRequiredFieldsWhenNotCopy(&newObj.Spec) {
		return nil, fmt.Errorf("a Book without copyOf must have spec.title, spec.price, and spec.genre set (non-zero)")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Book.
func (v *BookCustomValidator) ValidateDelete(_ context.Context, obj *bookstoreexamplecomv1.Book) (admission.Warnings, error) {
	booklog.Info("Validation for Book upon deletion", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}

func hasAtLeastOneOverride(spec *bookstoreexamplecomv1.BookSpec) bool {
	if spec.CopyOf == nil {
		return true
	}
	return spec.Title != "" || spec.Price != "" || spec.Genre != ""
}

func hasRequiredFieldsWhenNotCopy(spec *bookstoreexamplecomv1.BookSpec) bool {
	if spec.CopyOf != nil {
		return true
	}
	return spec.Title != "" && spec.Price != "" && spec.Genre != ""
}

func (v *BookCustomValidator) validateCopyOfReference(ctx context.Context, obj *bookstoreexamplecomv1.Book) error {
	booklog.Info("Validating spec.copyOf reference", "namespace", obj.GetNamespace(), "name", obj.GetName())
	if obj.Spec.CopyOf == nil {
		return nil
	}
	copyOf := obj.Spec.CopyOf
	booklog.Info("Validating spec.copyOf reference", "namespace", obj.GetNamespace(), "name", obj.GetName(),
		"copyOf.namespace", copyOf.Namespace, "copyOf.name", copyOf.Name)

	if obj.GetNamespace() == copyOf.Namespace && obj.GetName() == copyOf.Name {
		return fmt.Errorf("book cannot reference itself in spec.copyOf")
	}
	ref := bookstoreexamplecomv1.Book{}
	err := v.Reader.Get(ctx, types.NamespacedName{Namespace: copyOf.Namespace, Name: copyOf.Name}, &ref)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("spec.copyOf references non-existent Book")
		}
		return fmt.Errorf("failed to validate spec.copyOf reference")
	}
	if ref.Spec.CopyOf != nil {
		return fmt.Errorf("a copy cannot reference another copy, only originals can be copied")
	}
	return nil
}
