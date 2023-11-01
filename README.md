# ResourceConsist
## Motivation
### Goals
Making a customized controller can be realized easily, and offering the ability of controllers following PodOpsLifecycle.

The only thing users need to do to realize a customized controller is writing an adapter implementing ReconcileAdapter.
## Key Concepts
### Employer
**Employer** is the entity responsible for managing and coordinating the utilization of another resource, similar to how a service selects and controls pods.

Employer can be any kind, and CRD is of course can be used as Employer.
### Employee
**Employee** is the resource managed by another resource, like pods selected by service.

Same with Employer, Employee can be any kind, and CRD is of course can be used as Employee.

If an adapter implements ReconcileAdapter and follows PodOpsLifecycle, the Employee should be Pod.
## Key Interface/Struct Definitions
### ReconcileAdapter
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


// ReconcileLifecycleOptions defines whether PodOpsLifecycle followed and
// whether employees' LifecycleFinalizer conditions need to be Recorded/Erased to employer's anno.
// Actually NeedRecordEmployees only needed for those adapters that follow PodOpsLifecycle,
// in the case of employment relationship might change and resources in backend provider might be changed by others.
// If not implemented, the two options would be FollowPodOpsLifeCycle: true and NeedRecordEmployees: false
type ReconcileLifecycleOptions interface {
	FollowPodOpsLifeCycle() bool
	NeedRecordEmployees() bool
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

ReconcileOptions and ReconcileWatchOptions Interfaces can be optional implemented, dependent on whether customized controllers need specify some reconcile options like rate limiter.

Service/Pod will be default Employer/Employee, if ReconcileWatchOptions not implemented. And there is a default Predicate which filters out Services without Label: ```"kusionstack.io/control": "true"```.
### IEmployer/IEmployee
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
### PodEmployeeStatuses
**PodEmployeeStatuses** is a built-in struct implementing EmployeeStatus.EmployeeStatuses.
ExtraStatus in PodEmployeeStatuses is an interface so that adapters can implement it as they wished. Normally, ExtraStatus is extra info beyond basic pod status related to backend provider, like the traffic status of backend server(pod) under load balancer.
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
### PodAvailableConditions
Used if PodOpsLifecycle followed.

**PodAvailableConditions** is an annotation on pod, indicating what finalizer should be added to achieve service-available state.

For a pod employed by multiple employers, there will be multiple LifecycleFinalizer should be added, and PodAvailableConditions annotation will record what employers and what finalizers are.

Webhook will also record PodAvailableConditions in case of Pod creation to avoid Pod reaching service-available state if ResourceConsist controller not record PodAvailableConditions before Pod ready.
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
## Key Finalizers
### LifecycleFinalizer
**LifecycleFinalizer** prefixed with <mark>"prot.podopslifecycle.kusionstack.io"</mark>, is a finalizer on Employee used to follow PodOpsLifecycle, removed in preparing period of PodOpsLifecycle and added in completing period of PodOpsLifecycle
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
### CleanFinalizer
**CleanFinalizer** is a finalizer on Employer, used to bind Employer and Employee.

CleanFinalizer should be added in the first Reconcile of the resource, and be removed only when there is no more relation between Employer and Employee and during deletion.
```Go
cleanFinalizerPrefix = "resource-consist.kusionstack.io/clean-"
	
cleanFlz := cleanFinalizerPrefix + employer.GetName()
```
## Main Logic of Reconcile in ResourceConsist Controller
### Ensure clean finalizer of Employer
Clean finalizer will be added to Employer before everything, and all resources related to Employer will be cleaned before clean finalizer removed, so that nothing related to Employer will be remained.
### Ensure PodAvailableConditions if PodOpsLifecycle followed
An expected LifecycleFinalizer related to Employer will be added into PodAvailableConditions.
### Get Expect/Current Employer/Employee, make diff, and do sync
Adapters should implement these methods, and Resource Consist Controller will call it.
## Tutorials
**kusionstack.io/resourceconsit** mainly consists of frame, experimental/adapters and adapters.

The frame, ```kusionstack.io/resourceconsist/pkg/frame```, is used for adapters starting a controller, which handles Reconcile and Employer/Employees' spec&status. If you wrote an adapter in your own repo, you can import ```kusionstack.io/resourceconsist/pkg/frame/controller``` and ```kusionstack.io/resourceconsist/pkg/frame/webhook```, and call AddToMgr to start a controller.
```Go
import (
    controllerframe "kusionstack.io/resourceconsist/pkg/frame/controller"
    webhookframe "kusionstack.io/resourceconsist/pkg/frame/webhook"
)

func main() {
    controllerframe.AddToMgr(manager, yourOwnControllerAdapter)
    webhookframe.AddToMgr(manager, yourOwnWebhookAdapter)
}
```
### adapters
The adapters, ```kusionstack.io/resourceconsist/pkg/adapters```, consists of built-in adapters. You can start a controller with built-in adapters just calling AddBuiltinControllerAdaptersToMgr and AddBuiltinWebhookAdaptersToMgr, passing built-in adapters' names. Currently, an aliababacloudslb adapter has released. You can use it as follows:
```Go
import (
    "kusionstack.io/resourceconsist/pkg/adapters"
)

func main() {
    adapters.AddBuiltinControllerAdaptersToMgr(manager, []adapters.AdapterName{adapters.AdapterAlibabaCloudSlb})
    adapters.AddBuiltinWebhookAdaptersToMgr(manager, []adapters.AdapterName{adapters.AdapterAlibabaCloudSlb})
}
```
Built-in adapters can also be used like how frame used. You can call NewAdapter from a certain built-in adapter pkg and the call frame.AddToMgr to start a controller/webhook

More built-in adapters will be implemented in the future. To make this repo stable, all new built-in adapters will be added to ```kusionstack.io/pkg/experimental/adapters``` first, and then moved to ```kusionstack.io/pkg/adapters``` until ready to be released.
#### alibabacloudslb adapter
```pkg/adapters/alibabacloudslb``` is an adapter that implements ReconcileAdapter. It follows **PodOpsLifecycle** to handle various scenarios during pod operations, such as creating a new pod, deleting an existing pod, or handling changes to pod configurations. This adapter ensures minimal traffic loss and provides a seamless experience for users accessing services load balanced by Alibaba Cloud SLB.

In ```pkg/adapters/alibabacloudslb```, the real server is removed from SLB before pod operation in ACK. The LB management and real server management are handled by CCM in ACK. Since alibabacloudslb adapter follows PodOpsLifecycle and real servers are managed by CCM, ReconcileLifecycleOptions should be implemented. If the cluster is not in ACK or CCM is not working in the cluster, the alibabacloudslb controller should implement additional methods of ReconcileAdapter.
### experimental/adapters
The experimental/adapters is more like a pre-release pkg for built-in adapters. Usage of experimental/adapters is same with built-in adapters, and be aware that **DO NOT USE EXPERIMENTAL/ADAPTERS IN PRODUCTION**
### demo adapter
A demo is implemented in ```resource_controller_suite_test.go```. In the demo controller, the employer is represented as a service and is expected to have the following **DemoServiceStatus**:
```
DemoServiceStatus{
    EmployerId: employer.GetName(),
    EmployerStatuses: DemoServiceDetails{
        RemoteVIP:    "demo-remote-VIP",
        RemoteVIPQPS: 100,
    }
}
```
The employee is represented as a pod and is expected to have the following **DemoPodStatus**:
```
DemoPodStatus{
    EmployeeId:   pod.Name,
    EmployeeName: pod.Name,
    EmployeeStatuses: PodEmployeeStatuses{
        Ip: string,
        Ipv6: string,
        LifecycleReady: bool,
        ExtraStatus: PodExtraStatus{
            TrafficOn: bool,
            TrafficWeight: int,
        },
    }
}
```
The DemoResourceProviderClient is a fake client that handles backend provider resources related to the employer/employee (service/pods). In the Demo Controller, ```demoResourceVipStatusInProvider``` and ```demoResourceRsStatusInProvider``` are mocked as resources in the backend provider.

How the demo controller adapter realized will be introduced in detail as follows,
```DemoControllerAdapter``` was defined, including a kubernetes client and a resourceProviderClient. What included in the Adapter struct can be defined as needed.
```Go
type DemoControllerAdapter struct {
	client.Client
	resourceProviderClient *DemoResourceProviderClient
}
```
Declaring that the DemoControllerAdapter implemented ```ReconcileAdapter``` and ```ReconcileLifecycleOptions```. Implementing ```RconcileAdapter``` is a must action, while ```ReconcileLifecycleOptions``` isn't, check the remarks for ```ReconcileLifecycleOptions``` in ```kusionstack.io/resourceconsist/pkg/frame/controller/types.go``` to find why.
```Go
var _ ReconcileAdapter = &DemoControllerAdapter{}
var _ ReconcileLifecycleOptions = &DemoControllerAdapter{}
```
Following two methods for DemoControllerAdapter inplementing ```ReconcileLifecycleOptions```, defines whether DemoControllerAdapter following PodOpsLifecycle and need record employees.
```Go
func (r *DemoControllerAdapter) FollowPodOpsLifeCycle() bool {
	return true
}

func (r *DemoControllerAdapter) NeedRecordEmployees() bool {
	return needRecordEmployees
}
```
```IEmployer``` and ```IEmployee``` are interfaces that includes several methods indicating the status employer and employee.
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

type DemoServiceStatus struct {
	EmployerId       string
	EmployerStatuses DemoServiceDetails
}

type DemoServiceDetails struct {
	RemoteVIP    string
	RemoteVIPQPS int
}

type DemoPodStatus struct {
	EmployeeId       string
	EmployeeName     string
	EmployeeStatuses PodEmployeeStatuses
}
```
```GetSelectedEmployeeNames``` returns all employees' names selected by employer, here is pods' names selected by service. ```GetSelectedEmployeeNames``` is used for ensuring LifecycleFinalizer and ExpectedFinalizer, so you can give it an empty return if your adapter doesn't follow PodOpsLifecycle. 
```Go
func (r *DemoControllerAdapter) GetSelectedEmployeeNames(ctx context.Context, employer client.Object) ([]string, error) {
	svc, ok := employer.(*corev1.Service)
	if !ok {
		return nil, fmt.Errorf("expect employer kind is Service")
	}
	selector := labels.Set(svc.Spec.Selector).AsSelectorPreValidated()
	var podList corev1.PodList
	err := r.List(ctx, &podList, &client.ListOptions{Namespace: svc.Namespace, LabelSelector: selector})
	if err != nil {
		return nil, err
	}

	selected := make([]string, len(podList.Items))
	for idx, pod := range podList.Items {
		selected[idx] = pod.Name
	}

	return selected, nil
}
```
```GetExpectedEmployer``` and ```GetCurrentEmployer``` defines what is expected under the spec of employer and what is current status, like the load balancer from a cloud provider. Here in the demo adapter, expected is defined by hardcode and current is retrieved from a fake resource provider ```demoResourceVipStatusInProvider```.
```Go
func (r *DemoControllerAdapter) GetExpectedEmployer(ctx context.Context, employer client.Object) ([]IEmployer, error) {
	if !employer.GetDeletionTimestamp().IsZero() {
		return nil, nil
	}
	var expect []IEmployer
	expect = append(expect, DemoServiceStatus{
		EmployerId: employer.GetName(),
		EmployerStatuses: DemoServiceDetails{
			RemoteVIP:    "demo-remote-VIP",
			RemoteVIPQPS: 100,
		},
	})
	return expect, nil
}

func (r *DemoControllerAdapter) GetCurrentEmployer(ctx context.Context, employer client.Object) ([]IEmployer, error) {
	var current []IEmployer

	req := &DemoResourceVipOps{}
	resp, err := r.resourceProviderClient.QueryVip(req)
	if err != nil {
		return current, err
	}
	if resp == nil {
		return current, fmt.Errorf("demo resource vip query resp is nil")
	}

	for _, employerStatus := range resp.VipStatuses {
		current = append(current, employerStatus)
	}
	return current, nil
}
```
```CreateEmployer/UpdateEmployer/DeleteEmployer``` handles creation/update/deletion of resources related to employer on related backend provider. Here in the demo adapter, ```CreateEmployer/UpdateEmployer/DeleteEmployer``` handles ```demoResourceVipStatusInProvider```.
```Go
func (r *DemoControllerAdapter) CreateEmployer(ctx context.Context, employer client.Object, toCreates []IEmployer) ([]IEmployer, []IEmployer, error) {
	if toCreates == nil || len(toCreates) == 0 {
		return toCreates, nil, nil
	}

	toCreateDemoServiceStatus := make([]DemoServiceStatus, len(toCreates))
	for idx, create := range toCreates {
		createDemoServiceStatus, ok := create.(DemoServiceStatus)
		if !ok {
			return nil, toCreates, fmt.Errorf("toCreates employer is not DemoServiceStatus")
		}
		toCreateDemoServiceStatus[idx] = createDemoServiceStatus
	}

	_, err := r.resourceProviderClient.CreateVip(&DemoResourceVipOps{
		VipStatuses: toCreateDemoServiceStatus,
	})
	if err != nil {
		return nil, toCreates, err
	}
	return toCreates, nil, nil
}

func (r *DemoControllerAdapter) UpdateEmployer(ctx context.Context, employer client.Object, toUpdates []IEmployer) ([]IEmployer, []IEmployer, error) {
	if toUpdates == nil || len(toUpdates) == 0 {
		return toUpdates, nil, nil
	}

	toUpdateDemoServiceStatus := make([]DemoServiceStatus, len(toUpdates))
	for idx, update := range toUpdates {
		updateDemoServiceStatus, ok := update.(DemoServiceStatus)
		if !ok {
			return nil, toUpdates, fmt.Errorf("toUpdates employer is not DemoServiceStatus")
		}
		toUpdateDemoServiceStatus[idx] = updateDemoServiceStatus
	}

	_, err := r.resourceProviderClient.UpdateVip(&DemoResourceVipOps{
		VipStatuses: toUpdateDemoServiceStatus,
	})
	if err != nil {
		return nil, toUpdates, err
	}
	return toUpdates, nil, nil
}

func (r *DemoControllerAdapter) DeleteEmployer(ctx context.Context, employer client.Object, toDeletes []IEmployer) ([]IEmployer, []IEmployer, error) {
	if toDeletes == nil || len(toDeletes) == 0 {
		return toDeletes, nil, nil
	}

	toDeleteDemoServiceStatus := make([]DemoServiceStatus, len(toDeletes))
	for idx, update := range toDeletes {
		deleteDemoServiceStatus, ok := update.(DemoServiceStatus)
		if !ok {
			return nil, toDeletes, fmt.Errorf("toDeletes employer is not DemoServiceStatus")
		}
		toDeleteDemoServiceStatus[idx] = deleteDemoServiceStatus
	}

	_, err := r.resourceProviderClient.DeleteVip(&DemoResourceVipOps{
		VipStatuses: toDeleteDemoServiceStatus,
	})
	if err != nil {
		return nil, toDeletes, err
	}
	return toDeletes, nil, nil
}
```
```GetExpectedEmployee```and```GetCurrentEmployee``` defines what is expected under the spec of employer and employees and what is current status, like real servers under the load balancer from a cloud provider. Here in the demo adapter, expected is calculated from pods and current is retrieved from a fake resource provider ```demoResourceRsStatusInProvider```.
```Go
// GetExpectEmployeeStatus return expect employee status
func (r *DemoControllerAdapter) GetExpectedEmployee(ctx context.Context, employer client.Object) ([]IEmployee, error) {
	if !employer.GetDeletionTimestamp().IsZero() {
		return []IEmployee{}, nil
	}

	svc, ok := employer.(*corev1.Service)
	if !ok {
		return nil, fmt.Errorf("expect employer kind is Service")
	}
	selector := labels.Set(svc.Spec.Selector).AsSelectorPreValidated()

	var podList corev1.PodList
	err := r.List(ctx, &podList, &client.ListOptions{Namespace: svc.Namespace, LabelSelector: selector})
	if err != nil {
		return nil, err
	}

	expected := make([]IEmployee, len(podList.Items))
	expectIdx := 0
	for _, pod := range podList.Items {
		if !pod.DeletionTimestamp.IsZero() {
			continue
		}
		status := DemoPodStatus{
			EmployeeId:   pod.Name,
			EmployeeName: pod.Name,
		}
		employeeStatuses, err := GetCommonPodEmployeeStatus(&pod)
		if err != nil {
			return nil, err
		}
		extraStatus := PodExtraStatus{}
		if employeeStatuses.LifecycleReady {
			extraStatus.TrafficOn = true
			extraStatus.TrafficWeight = 100
		} else {
			extraStatus.TrafficOn = false
			extraStatus.TrafficWeight = 0
		}
		employeeStatuses.ExtraStatus = extraStatus
		status.EmployeeStatuses = employeeStatuses
		expected[expectIdx] = status
		expectIdx++
	}

	return expected[:expectIdx], nil
}

func (r *DemoControllerAdapter) GetCurrentEmployee(ctx context.Context, employer client.Object) ([]IEmployee, error) {
	var current []IEmployee
	req := &DemoResourceRsOps{}
	resp, err := r.resourceProviderClient.QueryRealServer(req)
	if err != nil {
		return current, err
	}
	if resp == nil {
		return current, fmt.Errorf("demo resource rs query resp is nil")
	}

	for _, rsStatus := range resp.RsStatuses {
		current = append(current, rsStatus)
	}
	return current, nil
}
```
```CreateEmployees/UpdateEmployees/DeleteEmployees``` handles creation/update/deletion of resources related to employee on related backend provider. Here in the demo adapter, ```CreateEmployees/UpdateEmployees/DeleteEmployees``` handles ```demoResourceRsStatusInProvider```.
```Go
func (r *DemoControllerAdapter) CreateEmployees(ctx context.Context, employer client.Object, toCreates []IEmployee) ([]IEmployee, []IEmployee, error) {
	if toCreates == nil || len(toCreates) == 0 {
		return toCreates, nil, nil
	}
	toCreateDemoPodStatuses := make([]DemoPodStatus, len(toCreates))

	for idx, toCreate := range toCreates {
		podStatus, ok := toCreate.(DemoPodStatus)
		if !ok {
			return nil, toCreates, fmt.Errorf("toCreate is not DemoPodStatus")
		}
		toCreateDemoPodStatuses[idx] = podStatus
	}

	_, err := r.resourceProviderClient.CreateRealServer(&DemoResourceRsOps{
		RsStatuses: toCreateDemoPodStatuses,
	})
	if err != nil {
		return nil, toCreates, err
	}

	return toCreates, nil, nil
}

func (r *DemoControllerAdapter) UpdateEmployees(ctx context.Context, employer client.Object, toUpdates []IEmployee) ([]IEmployee, []IEmployee, error) {
	if toUpdates == nil || len(toUpdates) == 0 {
		return toUpdates, nil, nil
	}

	toUpdateDemoPodStatuses := make([]DemoPodStatus, len(toUpdates))

	for idx, toUpdate := range toUpdates {
		podStatus, ok := toUpdate.(DemoPodStatus)
		if !ok {
			return nil, toUpdates, fmt.Errorf("toUpdate is not DemoPodStatus")
		}
		toUpdateDemoPodStatuses[idx] = podStatus
	}

	_, err := r.resourceProviderClient.UpdateRealServer(&DemoResourceRsOps{
		RsStatuses: toUpdateDemoPodStatuses,
	})
	if err != nil {
		return nil, toUpdates, err
	}

	return toUpdates, nil, nil
}

func (r *DemoControllerAdapter) DeleteEmployees(ctx context.Context, employer client.Object, toDeletes []IEmployee) ([]IEmployee, []IEmployee, error) {
	if toDeletes == nil || len(toDeletes) == 0 {
		return toDeletes, nil, nil
	}

	toDeleteDemoPodStatuses := make([]DemoPodStatus, len(toDeletes))

	for idx, toDelete := range toDeletes {
		podStatus, ok := toDelete.(DemoPodStatus)
		if !ok {
			return nil, toDeletes, fmt.Errorf("toDelete is not DemoPodStatus")
		}
		toDeleteDemoPodStatuses[idx] = podStatus
	}

	_, err := r.resourceProviderClient.DeleteRealServer(&DemoResourceRsOps{
		RsStatuses: toDeleteDemoPodStatuses,
	})
	if err != nil {
		return nil, toDeletes, err
	}

	return toDeletes, nil, nil
}
```
