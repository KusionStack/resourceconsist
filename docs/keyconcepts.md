# Key Concepts
## 🤠Employer
**Employer** is the entity responsible for managing and coordinating the utilization of another resource, 
similar to how a service selects and controls pods.

Employer can be any kind, and CRD is of course can be used as Employer.
## 👩‍💻Employee
**Employee** is the resource managed by another resource, like pods selected by service.

Same with Employer, Employee can be any kind, and CRD is of course can be used as Employee.

>If an adapter implements ReconcileAdapter and follows PodOpsLifecycle, the Employee should be Pod.
# ✨Key Interface/Struct Definitions
## ReconcileAdapter
**ReconcileAdapter** is an interface specifying a set of methods as follows.
```Go
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
```
A customized controller must realize an adapter implementing the ReconcileAdapter.

ReconcileOptions and ReconcileWatchOptions Interfaces can be optional implemented, dependent on whether customized 
controllers need specify some reconcile options like rate limiter.

>Service/Pod will be default Employer/Employee, if ReconcileWatchOptions not implemented. And there is a default 
>Predicate which filters out Services without Label: ```"kusionstack.io/control": "true"```.
## IEmployer/IEmployee
**IEmployer/IEmployee** are interfaces defined as follows.
```Go
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
```
## PodEmployeeStatuses
**PodEmployeeStatuses** is a built-in struct implementing EmployeeStatus.EmployeeStatuses.
ExtraStatus in PodEmployeeStatuses is an interface so that adapters can implement it as they wished. Normally, 
ExtraStatus is extra info beyond basic pod status related to backend provider, like the traffic status of backend 
server(pod) under load balancer.
```Go
type PodEmployeeStatuses struct {
	// can be set by calling SetCommonPodEmployeeStatus
	Ip             string `json:"ip,omitempty"`
	Ipv6           string `json:"ipv6,omitempty"`
	LifecycleReady bool   `json:"lifecycleReady,omitempty"`
	// extra info related to backend provider
	ExtraStatus interface{} `json:"extraStatus,omitempty"`
}
```
## PodAvailableConditions
Used if PodOpsLifecycle followed.

**PodAvailableConditions** is an annotation on pod, indicating what finalizer should be added to achieve 
[service-available state](https://kusionstack.io/docs/operating/concepts/podopslifecycle).

For a pod employed by multiple employers, there will be multiple LifecycleFinalizer should be added, and 
PodAvailableConditions annotation will record what employers and what finalizers are.

Webhook will also record PodAvailableConditions in case of Pod creation to avoid Pod reaching service-available 
state if ResourceConsist controller not record PodAvailableConditions before Pod ready.
```Go
const PodAvailableConditionsAnnotation = "pod.kusionstack.io/available-conditions" // indicate the available conditions of a pod

type PodAvailableConditions struct {
	ExpectedFinalizers map[string]string `json:"expectedFinalizers,omitempty"` // indicate the expected finalizers of a pod
}

func GenerateLifecycleFinalizerKey(employer client.Object) string {
	return fmt.Sprintf("%s/%s/%s", employer.GetObjectKind().GroupVersionKind().Kind,
		employer.GetNamespace(), employer.GetName())
}

func GenerateLifecycleFinalizer(employerName string) string {
	b := md5.Sum([]byte(employerName))
	return v1alpha1.PodOperationProtectionFinalizerPrefix + "/" + hex.EncodeToString(b[:])[8:24]
}
```
# ✨Key Finalizers
## LifecycleFinalizer
**LifecycleFinalizer** prefixed with ```prot.podopslifecycle.kusionstack.io```, is a finalizer on Employee used to 
follow PodOpsLifecycle, removed in preparing period of PodOpsLifecycle and added in completing period of PodOpsLifecycle
```Go
const (
	PodOperationProtectionFinalizerPrefix = "prot.podopslifecycle.kusionstack.io"
)

func GenerateLifecycleFinalizerKey(employer client.Object) string {
	return fmt.Sprintf("%s/%s/%s", employer.GetObjectKind().GroupVersionKind().Kind,
		employer.GetNamespace(), employer.GetName())
}

func GenerateLifecycleFinalizer(employerName string) string {
	b := md5.Sum([]byte(employerName))
	return v1alpha1.PodOperationProtectionFinalizerPrefix + "/" + hex.EncodeToString(b[:])[8:24]
}
```
## CleanFinalizer
**CleanFinalizer** is a finalizer on Employer, used to bind Employer and Employee.

CleanFinalizer should be added in the first Reconcile of the resource, and be removed only when there is no more 
relation between Employer and Employee and during deletion.
```Go
cleanFinalizerPrefix = "resource-consist.kusionstack.io/clean-"
	
cleanFlz := cleanFinalizerPrefix + employer.GetName()
```
