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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer"
	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer/platform"

	apimanifests "github.com/k8stopologyawareschedwg/deployer/pkg/manifests/api"
	rtemanifests "github.com/k8stopologyawareschedwg/deployer/pkg/manifests/rte"

	topologyexporterv1alpha1 "github.com/openshift-kni/rte-operator/api/v1alpha1"

	"github.com/openshift-kni/rte-operator/pkg/apply"
	apistate "github.com/openshift-kni/rte-operator/pkg/objectstate/api"
	"github.com/openshift-kni/rte-operator/pkg/objectstate/rte"
	rtestate "github.com/openshift-kni/rte-operator/pkg/objectstate/rte"
	"github.com/openshift-kni/rte-operator/pkg/status"
)

const (
	defaultResourceTopologyExporterCrName = "resourcetopologyexporter"
)

// ResourceTopologyExporterReconciler reconciles a ResourceTopologyExporter object
type ResourceTopologyExporterReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	Platform     platform.Platform
	APIManifests apimanifests.Manifests
	RTEManifests rtemanifests.Manifests
	Helper       *deployer.Helper
	Namespace    string
	ImageSpec    string
}

// TODO: narrow down

// Namespace Scoped
// TODO

// Cluster Scoped
//+kubebuilder:rbac:groups=topology.node.k8s.io,resources=noderesourcetopologies,verbs=get;list;create;update
//+kubebuilder:rbac:groups=config.openshift.io,resources=clusterversionss,verbs=list
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=topologyexporter.openshift-kni.io,resources=resourcetopologyexporters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=topologyexporter.openshift-kni.io,resources=resourcetopologyexporters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=topologyexporter.openshift-kni.io,resources=resourcetopologyexporters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *ResourceTopologyExporterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	logger := r.Log.WithValues("rte", req.NamespacedName)

	instance := &topologyexporterv1alpha1.ResourceTopologyExporter{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	if req.Name != defaultResourceTopologyExporterCrName {
		err := fmt.Errorf("ResourceTopologyExporter resource name must be %q", defaultResourceTopologyExporterCrName)
		logger.Error(err, "Incorrect ResourceTopologyExporter resource name", "name", req.Name)
		if err := status.Update(context.TODO(), r.Client, instance, status.ConditionDegraded, "IncorrectResourceTopologyExporterResourceName", fmt.Sprintf("Incorrect ResourceTopologyExporter resource name: %s", req.Name)); err != nil {
			logger.Error(err, "Failed to update resourcetopologyexporter status", "Desired status", status.ConditionDegraded)
		}
		return ctrl.Result{}, nil // Return success to avoid requeue
	}

	// note we intentionally NOT update the APIManifests - it is expected to be a NOP anyway
	if r.Namespace != req.NamespacedName.Namespace {
		r.RTEManifests = r.RenderManifests(req.NamespacedName.Namespace)
		r.Namespace = req.NamespacedName.Namespace
	}

	result, condition, err := r.reconcileResource(ctx, req, instance)
	if condition != "" {
		// TODO: use proper reason
		reason, message := condition, messageFromError(err)
		if err := status.Update(context.TODO(), r.Client, instance, condition, reason, message); err != nil {
			logger.Info("Failed to update resourcetopologyexporter status", "Desired condition", condition, "error", err)
		}
	}
	return result, err
}

// RenderManifests renders the reconciler manifests so they can be deployed on the cluster.
func (r *ResourceTopologyExporterReconciler) RenderManifests(namespace string) rtemanifests.Manifests {
	logger := r.Log.WithValues("rte", namespace)
	logger.Info("Updating manifests")
	mf := r.RTEManifests.Update(rtemanifests.UpdateOptions{
		Namespace: namespace,
	})
	rtestate.UpdateDaemonSetImage(mf.DaemonSet, r.ImageSpec)
	return mf
}

func messageFromError(err error) string {
	if err == nil {
		return ""
	}
	unwErr := errors.Unwrap(err)
	if unwErr == nil {
		return ""
	}
	return unwErr.Error()
}

func (r *ResourceTopologyExporterReconciler) reconcileResource(ctx context.Context, req ctrl.Request, instance *topologyexporterv1alpha1.ResourceTopologyExporter) (ctrl.Result, string, error) {
	var err error
	err = r.syncNodeResourceTopologyAPI(instance)
	if err != nil {
		return ctrl.Result{}, status.ConditionDegraded, errors.Wrapf(err, "FailedAPISync")
	}

	dsInfo, err := r.syncResourceTopologyExporterResources(instance)
	if err != nil {
		return ctrl.Result{}, status.ConditionDegraded, errors.Wrapf(err, "FailedRTESync")
	}
	ok, err := r.Helper.IsDaemonSetRunning(dsInfo.Namespace, dsInfo.Name)
	if err != nil {
		return ctrl.Result{}, status.ConditionDegraded, err
	}
	if !ok {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, status.ConditionProgressing, nil
	}

	instance.Status.DaemonSet = &dsInfo
	return ctrl.Result{}, status.ConditionAvailable, nil
}

func (r *ResourceTopologyExporterReconciler) syncNodeResourceTopologyAPI(instance *topologyexporterv1alpha1.ResourceTopologyExporter) error {
	logger := r.Log.WithName("APISync")
	logger.Info("Start")

	Existing := apistate.FromClient(context.TODO(), r.Client, r.Platform, r.APIManifests)

	for _, objState := range Existing.State(r.APIManifests) {
		if _, err := apply.ApplyObject(context.TODO(), logger, r.Client, objState); err != nil {
			return errors.Wrapf(err, "could not create %s", objState.Desired.GetObjectKind().GroupVersionKind().String())
		}
	}
	return nil
}

func (r *ResourceTopologyExporterReconciler) syncResourceTopologyExporterResources(instance *topologyexporterv1alpha1.ResourceTopologyExporter) (topologyexporterv1alpha1.NamespacedName, error) {
	logger := r.Log.WithName("RTESync")
	logger.Info("Start")

	Existing := rtestate.FromClient(context.TODO(), r.Client, r.Platform, r.RTEManifests)

	res := topologyexporterv1alpha1.NamespacedName{}
	for _, objState := range Existing.State(r.RTEManifests) {
		if err := controllerutil.SetControllerReference(instance, objState.Desired, r.Scheme); err != nil {
			return res, errors.Wrapf(err, "Failed to set controller reference to %s %s", objState.Desired.GetNamespace(), objState.Desired.GetName())
		}
		obj, err := apply.ApplyObject(context.TODO(), logger, r.Client, objState)
		if err != nil {
			return res, errors.Wrapf(err, "could not apply (%s) %s/%s", objState.Desired.GetObjectKind().GroupVersionKind(), objState.Desired.GetNamespace(), objState.Desired.GetName())
		}

		if nname, ok := rte.NamespacedNameFromObject(obj); ok {
			res = nname
		}
	}
	return res, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceTopologyExporterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&topologyexporterv1alpha1.ResourceTopologyExporter{}).
		Complete(r)
}
