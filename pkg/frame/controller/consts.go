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

const (
	defaultMaxConcurrentReconciles    = 5
	expectedFinalizerAddedAnnoKey     = "resource-consist.kusionstack.io/employees-expected-finalizer-added"
	lifecycleFinalizerRecordedAnnoKey = "resource-consist.kusionstack.io/employees-lifecycle-finalizer-recorded"
	cleanFinalizerPrefix              = "resource-consist.kusionstack.io/clean-"
)

// Event reason list
const (
	EnsureEmployerCleanFinalizerFailed  = "EnsureEmployerCleanFinalizerFailed"
	EnsureEmployerCleanFinalizerSucceed = "EnsureEmployerCleanFinalizerSucceed"
	EnsureExpectedFinalizerFailed       = "EnsureExpectedFinalizerFailed"
	SyncEmployerFailed                  = "SyncEmployerFailed"
	SyncEmployeesFailed                 = "SyncEmployeesFailed"
	CleanEmployerCleanFinalizerFailed   = "CleanEmployerCleanFinalizerFailed"
	CleanEmployerCleanFinalizerSucceed  = "CleanEmployerCleanFinalizerSucceed"
)
