/*
Copyright 2023 The KusionStack Authors.

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
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// AddToMgr creates a new Controller of specified reconcileAdapter and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func AddToMgr(mgr manager.Manager, adapter ReconcileAdapter) error {
	r := NewReconcile(mgr, adapter)

	// CreateEmployees a new controller
	maxConcurrentReconciles := defaultMaxConcurrentReconciles
	rateLimiter := workqueue.DefaultControllerRateLimiter()
	if reconcileOptions, ok := adapter.(ReconcileOptions); ok {
		maxConcurrentReconciles = reconcileOptions.GetMaxConcurrent()
		rateLimiter = reconcileOptions.GetRateLimiter()
	}
	c, err := controller.New(adapter.GetControllerName(), mgr, controller.Options{
		MaxConcurrentReconciles: maxConcurrentReconciles,
		Reconciler:              r,
		RateLimiter:             rateLimiter})
	if err != nil {
		return err
	}

	if watchOptions, ok := adapter.(ReconcileWatchOptions); ok {
		// Watch for changes to EmployerResources
		err = c.Watch(&source.Kind{
			Type: watchOptions.NewEmployer()},
			watchOptions.EmployerEventHandler(),
			watchOptions.EmployerPredicates())
		if err != nil {
			return err
		}

		// Watch for changes to EmployeeResources
		err = c.Watch(&source.Kind{
			Type: watchOptions.NewEmployee()},
			watchOptions.EmployeeEventHandler(),
			watchOptions.EmployeePredicates())
		if err != nil {
			return err
		}

		return nil
	}

	// Watch for changes to EmployerResources
	err = c.Watch(&source.Kind{
		Type: &corev1.Service{}},
		&EnqueueServiceWithRateLimit{},
		employerPredicates)
	if err != nil {
		return err
	}

	// Watch for changes to EmployeeResources
	err = c.Watch(&source.Kind{
		Type: &corev1.Pod{}},
		&EnqueueServiceByPod{
			c: mgr.GetClient(),
		},
		employeePredicates)
	if err != nil {
		return err
	}

	return nil
}

func NewReconcile(mgr manager.Manager, reconcileAdapter ReconcileAdapter) *Consist {
	recorder := mgr.GetEventRecorderFor(reconcileAdapter.GetControllerName())
	return &Consist{
		Client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		adapter:  reconcileAdapter,
		logger:   logf.Log.WithName(reconcileAdapter.GetControllerName()),
		recorder: recorder,
	}
}

type Consist struct {
	client.Client
	scheme   *runtime.Scheme
	logger   logr.Logger
	recorder record.EventRecorder
	adapter  ReconcileAdapter
}

func (r *Consist) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var employer client.Object
	if watchOptions, ok := r.adapter.(ReconcileWatchOptions); ok {
		employer = watchOptions.NewEmployer()
	} else {
		employer = &corev1.Service{}
	}
	err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: request.Namespace,
		Name:      request.Name,
	}, employer)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		r.logger.Error(err, "get employer failed")
		return reconcile.Result{}, err
	}

	// ensure employer-clean finalizer firstly, employer-clean finalizer should be cleaned at the end
	updated, err := r.ensureEmployerCleanFlz(ctx, employer)
	if err != nil {
		r.logger.Error(err, "add employer clean finalizer failed")
		r.recorder.Eventf(employer, corev1.EventTypeWarning, "ensureEmployerCleanFlzFailed",
			"add employer clean finalizer failed: %s", err.Error())
		return reconcile.Result{}, err
	}
	if updated {
		r.logger.Info("add employer clean finalizer")
		r.recorder.Eventf(employer, corev1.EventTypeNormal, "ensureEmployerCleanFlzSucceed",
			"add employer clean finalizer")
		return reconcile.Result{}, nil
	}

	isExpectedClean, err := r.ensureExpectedFinalizer(ctx, employer)
	if err != nil {
		r.logger.Error(err, "ensure employees expected finalizer failed")
		r.recorder.Eventf(employer, corev1.EventTypeWarning, "ensureExpectedFinalizerFailed",
			"ensure employees expected finalizer failed: %s", err.Error())
		return reconcile.Result{}, err
	}

	expectEmployer, err := r.adapter.GetExpectedEmployer(ctx, employer)
	if err != nil {
		r.logger.Error(err, "get expect employer failed")
		r.recorder.Eventf(employer, corev1.EventTypeWarning, "GetExpectEmployerFailed",
			"get expect employer status failed: %s", err.Error())
		return reconcile.Result{}, err
	}
	currentEmployer, err := r.adapter.GetCurrentEmployer(ctx, employer)
	if err != nil {
		r.logger.Error(err, "get current employer failed")
		r.recorder.Eventf(employer, corev1.EventTypeWarning, "GetCurrentEmployerFailed",
			"get current employer status failed: %s", err.Error())
		return reconcile.Result{}, err
	}
	isCleanEmployer, syncEmployerFailedExist, err := r.syncEmployer(ctx, employer, expectEmployer, currentEmployer)
	if err != nil {
		r.logger.Error(err, "sync employer status failed")
		r.recorder.Eventf(employer, corev1.EventTypeWarning, "syncEmployerFailed",
			"sync employer status failed: %s", err.Error())
		return reconcile.Result{}, err
	}

	expectEmployees, err := r.adapter.GetExpectedEmployee(ctx, employer)
	if err != nil {
		r.logger.Error(err, "get expect employees failed")
		r.recorder.Eventf(employer, corev1.EventTypeWarning, "GetExpectEmployeeFailed",
			"get expect employees status failed: %s", err.Error())
		return reconcile.Result{}, err
	}
	currentEmployees, err := r.adapter.GetCurrentEmployee(ctx, employer)
	if err != nil {
		r.logger.Error(err, "get current employees failed")
		r.recorder.Eventf(employer, corev1.EventTypeWarning, "GetCurrentEmployeeFailed",
			"get current employees status failed: %s", err.Error())
		return reconcile.Result{}, err
	}
	isCleanEmployee, syncEmployeeFailedExist, err := r.syncEmployees(ctx, employer, expectEmployees, currentEmployees)
	if err != nil {
		r.logger.Error(err, "sync employees status failed")
		r.recorder.Eventf(employer, corev1.EventTypeWarning, "syncEmployeesFailed",
			"sync employees status failed: %s", err.Error())
		return reconcile.Result{}, err
	}

	if isCleanEmployer && isCleanEmployee && isExpectedClean && !employer.GetDeletionTimestamp().IsZero() {
		err = r.cleanEmployerCleanFinalizer(ctx, employer)
		if err != nil {
			r.logger.Error(err, "clean employer clean-finalizer failed")
			r.recorder.Eventf(employer, corev1.EventTypeWarning, "cleanEmployerCleanFinalizerFailed",
				"clean employer clean-finalizer failed: %s", err.Error())
			return reconcile.Result{}, err
		}
	}

	if syncEmployerFailedExist || syncEmployeeFailedExist {
		r.recorder.Eventf(employer, corev1.EventTypeNormal, "ReconcileFailed", "employer or employees synced failed exist")
		requeueOptions, requeueOptionsImplemented := r.adapter.(ReconcileRequeueOptions)
		if requeueOptionsImplemented {
			return reconcile.Result{RequeueAfter: requeueOptions.EmployeeSyncRequeueInterval()}, nil
		}
		return reconcile.Result{}, fmt.Errorf("employer or employees synced failed exist")
	}

	r.recorder.Eventf(employer, corev1.EventTypeNormal, "ReconcileSucceed", "")
	return reconcile.Result{}, nil
}
