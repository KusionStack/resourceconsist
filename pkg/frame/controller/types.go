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
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/ratelimiter"
)

// ReconcileOptions includes max concurrent reconciles and rate limiter,
// max concurrent reconcile: 5 and DefaultControllerRateLimiter() will be used if ReconcileOptions not implemented.
type ReconcileOptions interface {
	GetRateLimiter() ratelimiter.RateLimiter
	GetMaxConcurrent() int
}

// ReconcileWatchOptions defines what employer and employee is and how controller watch
// default employer: Service, default employee: Pod
// Recommend:
// implement ReconcileWatchOptions if Employer resource might be reconciled by other controller,
// different Predicates make an employer won't be reconciled by more than one controller so that LifecycleFinalizer won't
// be solved incorrectly.
type ReconcileWatchOptions interface {
	NewEmployer() client.Object
	NewEmployee() client.Object
	EmployerEventHandler() handler.EventHandler
	EmployeeEventHandler() handler.EventHandler
	EmployerPredicates() predicate.Funcs
	EmployeePredicates() predicate.Funcs
}

type ExpectedFinalizerRecordOptions interface {
	// NeedRecordExpectedFinalizerCondition only needed for those adapters that follow PodOpsLifecycle,
	// in the case of employment relationship might change(like label/selector changes) and the compensation logic
	// of kusionstack.io/operating can't handle the changes.
	// in most cases, this option is not needed.
	NeedRecordExpectedFinalizerCondition() bool
}

// ReconcileLifecycleOptions defines whether PodOpsLifecycle followed
// and whether employees' LifecycleFinalizer conditions need to be Recorded/Erased to employer's anno.
// If not implemented, the default options would be:
// FollowPodOpsLifeCycle: true and NeedRecordLifecycleFinalizerCondition: false
type ReconcileLifecycleOptions interface {
	FollowPodOpsLifeCycle() bool

	// NeedRecordLifecycleFinalizerCondition only needed for those adapters that follow PodOpsLifecycle,
	// in the case of employment relationship might change and resources in backend provider might be changed by others.
	NeedRecordLifecycleFinalizerCondition() bool
}

type ReconcileRequeueOptions interface {
	// EmployeeSyncRequeueInterval returns requeue time interval if employee synced failed but no err
	EmployeeSyncRequeueInterval() time.Duration
}

// ReconcileAdapter is the interface that customized controllers should implement.
type ReconcileAdapter interface {
	GetControllerName() string

	GetSelectedEmployeeNames(ctx context.Context, employer client.Object) ([]string, error)

	// GetExpectedEmployer and GetCurrentEmployer return expect/current status of employer from related backend provider
	GetExpectedEmployer(ctx context.Context, employer client.Object) ([]IEmployer, error)
	GetCurrentEmployer(ctx context.Context, employer client.Object) ([]IEmployer, error)

	// CreateEmployer/UpdateEmployer/DeleteEmployer handles creation/update/deletion of resources related to employer on related backend provider
	CreateEmployer(ctx context.Context, employer client.Object, toCreates []IEmployer) ([]IEmployer, []IEmployer, error)
	UpdateEmployer(ctx context.Context, employer client.Object, toUpdates []IEmployer) ([]IEmployer, []IEmployer, error)
	DeleteEmployer(ctx context.Context, employer client.Object, toDeletes []IEmployer) ([]IEmployer, []IEmployer, error)

	// GetExpectedEmployee and GetCurrentEmployee return expect/current status of employees from related backend provider
	GetExpectedEmployee(ctx context.Context, employer client.Object) ([]IEmployee, error)
	GetCurrentEmployee(ctx context.Context, employer client.Object) ([]IEmployee, error)

	// CreateEmployees/UpdateEmployees/DeleteEmployees handles creation/update/deletion of resources related to employee on related backend provider
	CreateEmployees(ctx context.Context, employer client.Object, toCreates []IEmployee) ([]IEmployee, []IEmployee, error)
	UpdateEmployees(ctx context.Context, employer client.Object, toUpdates []IEmployee) ([]IEmployee, []IEmployee, error)
	DeleteEmployees(ctx context.Context, employer client.Object, toDeletes []IEmployee) ([]IEmployee, []IEmployee, error)
}

type IEmployer interface {
	GetEmployerId() string
	GetEmployerStatuses() interface{}
	EmployerEqual(employer IEmployer) (bool, error)
}

type IEmployee interface {
	GetEmployeeId() string
	GetEmployeeName() string
	GetEmployeeStatuses() interface{}
	EmployeeEqual(employee IEmployee) (bool, error)
}

type ToCUDEmployer struct {
	ToCreate  []IEmployer
	ToUpdate  []IEmployer
	ToDelete  []IEmployer
	Unchanged []IEmployer
}

type ToCUDEmployees struct {
	ToCreate  []IEmployee
	ToUpdate  []IEmployee
	ToDelete  []IEmployee
	Unchanged []IEmployee
}

type PodEmployeeStatuses struct {
	// can be set by calling SetCommonPodEmployeeStatus
	Ip             string `json:"ip,omitempty"`
	Ipv6           string `json:"ipv6,omitempty"`
	LifecycleReady bool   `json:"lifecycleReady,omitempty"`
	// extra info related to backend provider
	ExtraStatus interface{} `json:"extraStatus,omitempty"`
}

type PodExpectedFinalizerOps struct {
	Name    string
	Succeed bool
}
