/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2021 Red Hat, Inc.
 */

package compare

import (
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/equality"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrWrongObjectType    = fmt.Errorf("given object does not match the comparator")
	ErrMismatchingObjects = fmt.Errorf("given objects have mismatching types")
)

func Object(existing, obj client.Object) (bool, error) {
	return equality.Semantic.DeepEqual(existing, obj), nil
}

func ServiceAccount(existing, obj client.Object) (bool, error) {
	// TODO
	return Object(existing, obj)
}

func CustomResourceDefinition(existing, obj client.Object) (bool, error) {
	objCrd, ok := obj.(*apiextensionsv1.CustomResourceDefinition)
	if !ok {
		return false, ErrWrongObjectType
	}
	existingCrd, ok := existing.(*apiextensionsv1.CustomResourceDefinition)
	if !ok {
		return false, ErrMismatchingObjects
	}
	return equality.Semantic.DeepEqual(existingCrd.Spec, objCrd.Spec), nil
}
