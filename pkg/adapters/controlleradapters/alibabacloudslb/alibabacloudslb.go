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

package alibabacloudslb

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"kusionstack.io/resourceconsist/pkg/controller_frame"
)

var _ controller_frame.ReconcileAdapter = &SlbControllerAdapter{}
var _ controller_frame.ReconcileLifecycleOptions = &SlbControllerAdapter{}

type SlbControllerAdapter struct {
	client.Client
	slbClient *AlibabaCloudSlbClient
}

func NewReconcileAdapter(c client.Client) (controller_frame.ReconcileAdapter, error) {
	slbClient, err := NewAlibabaCloudSlbClient()
	if err != nil {
		return nil, err
	}
	if slbClient == nil {
		return nil, fmt.Errorf("alibaba cloud slb client is nil")
	}

	return &SlbControllerAdapter{
		Client:    c,
		slbClient: slbClient,
	}, nil
}

func (r *SlbControllerAdapter) FollowPodOpsLifeCycle() bool {
	return true
}

func (r *SlbControllerAdapter) NeedRecordEmployees() bool {
	return true
}

func (r *SlbControllerAdapter) GetControllerName() string {
	return "alibaba-cloud-slb-controller"
}

func (r *SlbControllerAdapter) GetExpectedEmployer(ctx context.Context, employer client.Object) ([]controller_frame.IEmployer, error) {
	return nil, nil
}

func (r *SlbControllerAdapter) GetSelectedEmployeeNames(ctx context.Context, employer client.Object) ([]string, error) {
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

func (r *SlbControllerAdapter) GetCurrentEmployer(ctx context.Context, employer client.Object) ([]controller_frame.IEmployer, error) {
	return nil, nil
}

func (r *SlbControllerAdapter) CreateEmployer(ctx context.Context, employer client.Object, toCreates []controller_frame.IEmployer) ([]controller_frame.IEmployer, []controller_frame.IEmployer, error) {
	return nil, nil, nil
}

func (r *SlbControllerAdapter) UpdateEmployer(ctx context.Context, employer client.Object, toUpdates []controller_frame.IEmployer) ([]controller_frame.IEmployer, []controller_frame.IEmployer, error) {
	return nil, nil, nil
}

func (r *SlbControllerAdapter) DeleteEmployer(ctx context.Context, employer client.Object, toDeletes []controller_frame.IEmployer) ([]controller_frame.IEmployer, []controller_frame.IEmployer, error) {
	return nil, nil, nil
}

func (r *SlbControllerAdapter) GetExpectedEmployee(ctx context.Context, employer client.Object) ([]controller_frame.IEmployee, error) {
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

	expected := make([]controller_frame.IEmployee, len(podList.Items))
	for idx, pod := range podList.Items {
		status := AlibabaSlbPodStatus{
			EmployeeID:   pod.Status.PodIP,
			EmployeeName: pod.Name,
		}
		employeeStatuses, err := controller_frame.GetCommonPodEmployeeStatus(&pod)
		if err != nil {
			return nil, err
		}
		extraStatus := PodExtraStatus{}
		if employeeStatuses.LifecycleReady {
			extraStatus.TrafficOn = true
		} else {
			extraStatus.TrafficOn = false
		}
		employeeStatuses.ExtraStatus = extraStatus
		status.EmployeeStatuses = employeeStatuses
		expected[idx] = status
	}

	return expected, nil
}

func (r *SlbControllerAdapter) GetCurrentEmployee(ctx context.Context, employer client.Object) ([]controller_frame.IEmployee, error) {
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

	lbID := svc.GetLabels()[alibabaCloudSlbLbIdLabelKey]
	bsExistUnderSlb := make(map[string]bool)
	if lbID != "" {
		backendServers, err := r.slbClient.GetBackendServers(lbID)
		if err != nil {
			return nil, fmt.Errorf("get backend servers of slb failed, err: %s", err.Error())
		}
		for _, bs := range backendServers {
			bsExistUnderSlb[bs] = true
		}
	}

	current := make([]controller_frame.IEmployee, len(podList.Items))
	for idx, pod := range podList.Items {
		status := AlibabaSlbPodStatus{
			EmployeeID:   pod.Status.PodIP,
			EmployeeName: pod.Name,
		}
		employeeStatuses, err := controller_frame.GetCommonPodEmployeeStatus(&pod)
		if err != nil {
			return nil, err
		}
		extraStatus := PodExtraStatus{}
		if !bsExistUnderSlb[status.EmployeeID] {
			extraStatus.TrafficOn = false
		} else {
			extraStatus.TrafficOn = true
		}
		employeeStatuses.ExtraStatus = extraStatus
		status.EmployeeStatuses = employeeStatuses
		current[idx] = status
	}

	return current, nil
}

// CreateEmployees returns (nil, toCreate, nil) since CCM of ACK will sync bs of slb
func (r *SlbControllerAdapter) CreateEmployees(ctx context.Context, employer client.Object, toCreates []controller_frame.IEmployee) ([]controller_frame.IEmployee, []controller_frame.IEmployee, error) {
	return nil, toCreates, nil
}

// UpdateEmployees returns (nil, toUpdate, nil) since CCM of ACK will sync bs of slb
func (r *SlbControllerAdapter) UpdateEmployees(ctx context.Context, employer client.Object, toUpdates []controller_frame.IEmployee) ([]controller_frame.IEmployee, []controller_frame.IEmployee, error) {
	return nil, toUpdates, nil
}

// DeleteEmployees returns (nil, toDelete, nil) since CCM of ACK will sync bs of slb
func (r *SlbControllerAdapter) DeleteEmployees(ctx context.Context, employer client.Object, toDeletes []controller_frame.IEmployee) ([]controller_frame.IEmployee, []controller_frame.IEmployee, error) {
	return nil, toDeletes, nil
}
