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

package apply

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func getExistingObject(ctx context.Context, log logr.Logger, client k8sclient.Client, obj client.Object) (client.Object, string, error) {
	name := obj.GetName()
	namespace := obj.GetNamespace()
	if name == "" {
		return nil, "", fmt.Errorf("Object %s has no name", obj.GetObjectKind().GroupVersionKind().String())
	}
	gvk := obj.GetObjectKind().GroupVersionKind()
	// used for logging and errors
	objDesc := fmt.Sprintf("(%s) %s/%s", gvk.String(), namespace, name)
	log.Info("reconciling", "object", objDesc)

	// TODO: explain why we can't use client.Object
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(gvk)
	err := client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, existing)
	return existing, objDesc, err
}

// ApplyObject applies the desired object against the apiserver,
// merging it with any existing objects if already present.
func ApplyObject(ctx context.Context, log logr.Logger, client k8sclient.Client, obj client.Object) error {
	existing, objDesc, err := getExistingObject(ctx, log, client, obj)

	if err != nil && apierrors.IsNotFound(err) {
		log.Info("creating", "object", objDesc)
		err := client.Create(ctx, obj)
		if err != nil {
			return err
		}
		log.Info("created", "object", objDesc)
	}

	if existing == nil {
		return nil
	}

	// Merge the desired object with what actually exists
	if err := MergeObjectForUpdate(existing, obj); err != nil {
		return errors.Wrapf(err, "could not merge object %s with existing", objDesc)
	}
	if !equality.Semantic.DeepEqual(existing, obj) {
		log.Info("updating", "object", objDesc)
		if err := client.Update(ctx, obj); err != nil {
			return errors.Wrapf(err, "could not update object %s", objDesc)
		}
		log.Info("updated", "object", objDesc)
	}

	return nil
}

func IsClusterScoped(obj client.Object) bool {
	if _, ok := obj.(*corev1.ServiceAccount); ok {
		return true
	}
	return false
}
