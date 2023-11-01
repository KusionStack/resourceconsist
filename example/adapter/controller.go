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

package adapter

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	controllerframe "kusionstack.io/resourceconsist/pkg/frame/controller"
)

type ExampleControllerAdapter struct{}

var _ controllerframe.ReconcileAdapter = &ExampleControllerAdapter{}

func NewExampleControllerAdapter() *ExampleControllerAdapter {
	return &ExampleControllerAdapter{}
}

func (e *ExampleControllerAdapter) GetControllerName() string {
	return "resource-consist-example"
}

func (e *ExampleControllerAdapter) GetSelectedEmployeeNames(ctx context.Context, employer client.Object) ([]string, error) {
	return nil, nil
}

func (e *ExampleControllerAdapter) GetExpectedEmployer(ctx context.Context, employer client.Object) ([]controllerframe.IEmployer, error) {
	return nil, nil
}

func (e *ExampleControllerAdapter) GetCurrentEmployer(ctx context.Context, employer client.Object) ([]controllerframe.IEmployer, error) {
	return nil, nil
}

func (e *ExampleControllerAdapter) CreateEmployer(ctx context.Context, employer client.Object, toCreates []controllerframe.IEmployer) ([]controllerframe.IEmployer, []controllerframe.IEmployer, error) {
	return nil, nil, nil

}

func (e *ExampleControllerAdapter) UpdateEmployer(ctx context.Context, employer client.Object, toUpdates []controllerframe.IEmployer) ([]controllerframe.IEmployer, []controllerframe.IEmployer, error) {
	return nil, nil, nil
}

func (e *ExampleControllerAdapter) DeleteEmployer(ctx context.Context, employer client.Object, toDeletes []controllerframe.IEmployer) ([]controllerframe.IEmployer, []controllerframe.IEmployer, error) {
	return nil, nil, nil
}

func (e *ExampleControllerAdapter) GetExpectedEmployee(ctx context.Context, employer client.Object) ([]controllerframe.IEmployee, error) {
	return nil, nil
}

func (e *ExampleControllerAdapter) GetCurrentEmployee(ctx context.Context, employer client.Object) ([]controllerframe.IEmployee, error) {
	return nil, nil
}

func (e *ExampleControllerAdapter) CreateEmployees(ctx context.Context, employer client.Object, toCreates []controllerframe.IEmployee) ([]controllerframe.IEmployee, []controllerframe.IEmployee, error) {
	return nil, nil, nil
}

func (e *ExampleControllerAdapter) UpdateEmployees(ctx context.Context, employer client.Object, toUpdates []controllerframe.IEmployee) ([]controllerframe.IEmployee, []controllerframe.IEmployee, error) {
	return nil, nil, nil
}

func (e *ExampleControllerAdapter) DeleteEmployees(ctx context.Context, employer client.Object, toDeletes []controllerframe.IEmployee) ([]controllerframe.IEmployee, []controllerframe.IEmployee, error) {
	return nil, nil, nil
}
