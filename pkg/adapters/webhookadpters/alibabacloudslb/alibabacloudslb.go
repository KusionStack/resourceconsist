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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"kusionstack.io/kube-api/apps/v1alpha1"
	"kusionstack.io/resourceconsist/pkg/webhook_frame"
)

var _ webhook_frame.WebhookAdapter = &SlbWebhookAdapter{}

type SlbWebhookAdapter struct {
}

func NewWebhookAdapter() webhook_frame.WebhookAdapter {
	return &SlbWebhookAdapter{}
}

func (r *SlbWebhookAdapter) Name() string {
	return "alibaba-cloud-slb--webhook"
}

func (r *SlbWebhookAdapter) GetEmployersByEmployee(ctx context.Context, employee client.Object, c client.Client) ([]client.Object, error) {
	var employers []client.Object
	var err error

	serviceList := &corev1.ServiceList{}
	err = c.List(ctx, serviceList, client.InNamespace(employee.GetNamespace()))
	if err != nil {
		if apierrors.IsNotFound(err) {
			return employers, nil
		}
		return employers, err
	}

	for _, service := range serviceList.Items {
		if service.GetLabels()[v1alpha1.ControlledByKusionStackLabelKey] != "true" {
			continue
		}
		if labels.SelectorFromSet(service.Spec.Selector).Matches(labels.Set(employee.GetLabels())) {
			employers = append(employers, service.DeepCopy())
		}
	}

	return employers, nil
}
