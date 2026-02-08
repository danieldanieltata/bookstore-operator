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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	bookstoreexamplecomv1 "github.com/danieldanieltata/bookstore-operator/api/v1"
)

func testScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = bookstoreexamplecomv1.AddToScheme(s)
	return s
}

func TestValidateCreate_RejectsMissingRequiredFields(t *testing.T) {
	v := BookCustomValidator{}
	obj := &bookstoreexamplecomv1.Book{}
	obj.Spec.Title = ""
	obj.Spec.Price = ""
	obj.Spec.Genre = ""

	_, err := v.ValidateCreate(context.Background(), obj)
	if err == nil {
		t.Fatal("expected error when required fields missing")
	}
	if msg := err.Error(); msg != "a Book without copyOf must have spec.title, spec.price, and spec.genre set (non-zero)" {
		t.Errorf("unexpected error: %s", msg)
	}
}

func TestValidateCreate_AllowsValidBook(t *testing.T) {
	v := BookCustomValidator{}
	obj := &bookstoreexamplecomv1.Book{}
	obj.Spec.Title = "The Book"
	obj.Spec.Price = "10"
	obj.Spec.Genre = "Fiction"

	_, err := v.ValidateCreate(context.Background(), obj)
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
}

func TestValidateCreate_RejectsSelfReference(t *testing.T) {
	v := BookCustomValidator{}
	obj := &bookstoreexamplecomv1.Book{}
	obj.SetNamespace("default")
	obj.SetName("mybook")
	obj.Spec.CopyOf = &bookstoreexamplecomv1.CopyOf{Namespace: "default", Name: "mybook"}

	_, err := v.ValidateCreate(context.Background(), obj)
	if err == nil {
		t.Fatal("expected error for self-reference")
	}
	if msg := err.Error(); msg != "book cannot reference itself in spec.copyOf" {
		t.Errorf("unexpected error: %s", msg)
	}
}

func TestValidateCreate_CopyOfScenarios(t *testing.T) {
	original := &bookstoreexamplecomv1.Book{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "original"},
		Spec:       bookstoreexamplecomv1.BookSpec{Title: "Orig", Price: "1", Genre: "X"},
	}
	copyBook := &bookstoreexamplecomv1.Book{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "copy"},
		Spec: bookstoreexamplecomv1.BookSpec{
			Title: "Copy", Price: "2", Genre: "Y",
			CopyOf: &bookstoreexamplecomv1.CopyOf{Namespace: "default", Name: "original"},
		},
	}

	t.Run("rejects nonexistent reference", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(testScheme()).Build()
		v := BookCustomValidator{Client: c}
		obj := &bookstoreexamplecomv1.Book{}
		obj.SetNamespace("default")
		obj.SetName("new")
		obj.Spec.CopyOf = &bookstoreexamplecomv1.CopyOf{Namespace: "default", Name: "nonexistent"}

		_, err := v.ValidateCreate(context.Background(), obj)
		if err == nil {
			t.Fatal("expected error for nonexistent copyOf")
		}
		if msg := err.Error(); msg != "spec.copyOf references non-existent Book" {
			t.Errorf("unexpected error: %s", msg)
		}
	})

	t.Run("rejects copy-of-copy", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(original, copyBook).Build()
		v := BookCustomValidator{Client: c}
		obj := &bookstoreexamplecomv1.Book{}
		obj.SetNamespace("default")
		obj.SetName("new")
		obj.Spec.CopyOf = &bookstoreexamplecomv1.CopyOf{Namespace: "default", Name: "copy"}
		obj.Spec.Title = "X"

		_, err := v.ValidateCreate(context.Background(), obj)
		if err == nil {
			t.Fatal("expected error when copyOf points to a copy")
		}
		if msg := err.Error(); msg != "a copy cannot reference another copy, only originals can be copied" {
			t.Errorf("unexpected error: %s", msg)
		}
	})

	t.Run("rejects copyOf without override", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(original).Build()
		v := BookCustomValidator{Client: c}
		obj := &bookstoreexamplecomv1.Book{}
		obj.SetNamespace("default")
		obj.SetName("new")
		obj.Spec.CopyOf = &bookstoreexamplecomv1.CopyOf{Namespace: "default", Name: "original"}

		_, err := v.ValidateCreate(context.Background(), obj)
		if err == nil {
			t.Fatal("expected error when copyOf has no override")
		}
		if msg := err.Error(); msg != "a Book with copyOf must override title, price, or genre in spec" {
			t.Errorf("unexpected error: %s", msg)
		}
	})

	t.Run("allows copyOf with override", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(original).Build()
		v := BookCustomValidator{Reader: c}
		obj := &bookstoreexamplecomv1.Book{}
		obj.SetNamespace("default")
		obj.SetName("new")
		obj.Spec.CopyOf = &bookstoreexamplecomv1.CopyOf{Namespace: "default", Name: "original"}
		obj.Spec.Title = "New Title"

		_, err := v.ValidateCreate(context.Background(), obj)
		if err != nil {
			t.Fatalf("expected no error: %v", err)
		}
	})
}

func TestValidateUpdate_RejectsMissingRequiredFields(t *testing.T) {
	v := BookCustomValidator{}
	oldObj := &bookstoreexamplecomv1.Book{}
	oldObj.Spec.Title = "Old"
	oldObj.Spec.Price = "5"
	oldObj.Spec.Genre = "Drama"
	newObj := &bookstoreexamplecomv1.Book{}
	newObj.Spec.Title = ""
	newObj.Spec.Price = ""
	newObj.Spec.Genre = ""

	_, err := v.ValidateUpdate(context.Background(), oldObj, newObj)
	if err == nil {
		t.Fatal("expected error when required fields missing")
	}
	if msg := err.Error(); msg != "a Book without copyOf must have spec.title, spec.price, and spec.genre set (non-zero)" {
		t.Errorf("unexpected error: %s", msg)
	}
}
