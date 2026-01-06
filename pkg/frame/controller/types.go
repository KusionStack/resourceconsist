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

// MultiClusterOptions defines whether employee is under fed cluster
// "kusionstack.io/kube-utils/multicluster" is the solution we use for multi cluster
// if MultiClusterOptions implemented, the cache and client of manager should be generated via  "kusionstack.io/kube-utils/multicluster"
type MultiClusterOptions interface {
	// Employer should be under fed, otherwise, just forget multi cluster :)
	// EmployerFed() bool

	EmployeeFed() bool
}

type ExpectedFinalizerRecordOptions interface {
	// NeedRecordExpectedFinalizerCondition only needed for those adapters that follow PodOpsLifecycle,
	// in the case of employment relationship might change(like label/selector changes) and the compensation logic
	// of kusionstack.io/operating can't handle the changes.
	// in most cases, this option is not needed.
	NeedRecordExpectedFinalizerCondition() bool
}

// StatusRecordOptions defines methods of record something for adapters
type StatusRecordOptions interface {
	// RecordStatuses records statuses of employer and employees, called at the end of successful reconcile.
	// cudEmployerResults and cudEmployeeResults are the results of create/update/delete operations on employer and employees
	RecordStatuses(ctx context.Context, employer client.Object,
		cudEmployerResults CUDEmployerResults, cudEmployeeResults CUDEmployeeResults) error

	// RecordErrorConditions records error conditions, called at the end of failed reconcile.
	// usually, something recorded in RecordErrorConditions should be erased in RecordStatuses.
	RecordErrorConditions(ctx context.Context, employer client.Object, err error) error
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

	// GetSelectedEmployeeNames returns employees' names selected by employer
	// note: in multi cluster case, if adapters deployed in fed and employees are under local, the format of employeeName
	// should be "employeeName" + "#" + "clusterName"
	GetSelectedEmployeeNames(ctx context.Context, employer client.Object) ([]string, error)
}

type ReconcileRequeueOptions interface {
	// EmployeeSyncRequeueInterval returns requeue time interval if employee synced failed but no err
	EmployeeSyncRequeueInterval() time.Duration
}

// ReconcileAdapter is the interface that customized controllers should implement.
type ReconcileAdapter interface {
	GetControllerName() string

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
	SetEmployerStatuses(employerStatuses interface{})
	EmployerEqual(employer IEmployer) (bool, error)
}

type IEmployee interface {
	GetEmployeeId() string
	// GetEmployeeName returns employee's name
	// note: in multi cluster case, if adapters deployed in fed and employees are under local, the format of employeeName
	// should be "employeeName" + "#" + "clusterName"
	// GetEmployeeName need to be implemented if follow Lifecycle
	GetEmployeeName() string
	GetEmployeeStatuses() interface{}
	SetEmployeeStatuses(employeeStatuses interface{})
	EmployeeEqual(employee IEmployee) (bool, error)
}

type ToCUDEmployer struct {
	ToCreate  []IEmployer
	ToUpdate  []IEmployer
	ToDelete  []IEmployer
	Unchanged []IEmployer
}

type CUDEmployerResults struct {
	SuccCreated []IEmployer
	FailCreated []IEmployer
	SuccUpdated []IEmployer
	FailUpdated []IEmployer
	SuccDeleted []IEmployer
	FailDeleted []IEmployer
	Unchanged   []IEmployer
}

type ToCUDEmployees struct {
	ToCreate  []IEmployee
	ToUpdate  []IEmployee
	ToDelete  []IEmployee
	Unchanged []IEmployee
}

type CUDEmployeeResults struct {
	SuccCreated []IEmployee
	FailCreated []IEmployee
	SuccUpdated []IEmployee
	FailUpdated []IEmployee
	SuccDeleted []IEmployee
	FailDeleted []IEmployee
	Unchanged   []IEmployee
}

type PodEmployeeStatuses struct {
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
