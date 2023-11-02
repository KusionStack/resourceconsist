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

package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"kusionstack.io/kube-api/apps/v1alpha1"
	"kusionstack.io/resourceconsist/pkg/utils"
)

// AddToMgr is only necessary for controllers following PodOpsLifecycle
func AddToMgr(mgr manager.Manager, adapter WebhookAdapter) error {
	server := mgr.GetWebhookServer()
	logger := mgr.GetLogger().WithName("webhook")

	if len(adapter.Name()) == 0 {
		logger.Info("Skip registering handlers without a name")
		return nil
	}

	path := adapter.Name()
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	decoder, _ := admission.NewDecoder(mgr.GetScheme())
	server.Register(path, &webhook.Admission{Handler: NewPodResourceConsistWebhook(mgr.GetClient(), decoder, adapter)})
	logger.V(3).Info("Registered webhook handler", "path", path)

	return nil
}

var _ admission.Handler = &PodResourceConsistWebhook{}

func (r *PodResourceConsistWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.DryRun != nil && *req.DryRun {
		return admission.Allowed("dry run")
	}
	if req.Kind.Kind != "Pod" {
		return admission.Patched("NoMutating")
	}

	pod := &corev1.Pod{}
	err := r.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	err = r.Mutating(ctx, pod, req.Operation)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	marshalled, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.AdmissionRequest.Object.Raw, marshalled)
}

type PodResourceConsistWebhook struct {
	WebhookAdapter
	client.Client
	*admission.Decoder
}

func NewPodResourceConsistWebhook(cli client.Client, decoder *admission.Decoder, adapter WebhookAdapter) *PodResourceConsistWebhook {
	return &PodResourceConsistWebhook{
		adapter,
		cli,
		decoder,
	}
}

func (r *PodResourceConsistWebhook) Mutating(ctx context.Context, newPod *corev1.Pod, operation admissionv1.Operation) error {
	if newPod == nil {
		return nil
	}

	// only concern pods new created
	if operation != admissionv1.Create {
		return nil
	}

	employers, err := r.WebhookAdapter.GetEmployersByEmployee(ctx, newPod, r.Client)
	if err != nil {
		return err
	}

	availableExpectedFlzs := v1alpha1.PodAvailableConditions{
		ExpectedFinalizers: map[string]string{},
	}
	for _, employer := range employers {
		expectedFlzKey := utils.GenerateLifecycleFinalizerKey(employer)
		expectedFlz := utils.GenerateLifecycleFinalizer(employer.GetName())
		availableExpectedFlzs.ExpectedFinalizers[expectedFlzKey] = expectedFlz
	}
	annoAvailableCondition, err := json.Marshal(availableExpectedFlzs)
	if err != nil {
		return err
	}
	if newPod.Annotations == nil {
		newPod.Annotations = make(map[string]string)
	}
	newPod.Annotations[v1alpha1.PodAvailableConditionsAnnotation] = string(annoAvailableCondition)

	return nil
}
