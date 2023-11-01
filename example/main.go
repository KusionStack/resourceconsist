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

package main

import (
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	exampleadapter "kusionstack.io/resourceconsist/example/adapter"
	controllerframe "kusionstack.io/resourceconsist/pkg/frame/controller"
	webhookframe "kusionstack.io/resourceconsist/pkg/frame/webhook"
)

func main() {
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		os.Exit(1)
	}

	exampleControllerAdapter := exampleadapter.NewExampleControllerAdapter()

	exampleWebhookAdapter := exampleadapter.NewExampleWebhookAdapter()

	err = controllerframe.AddToMgr(mgr, exampleControllerAdapter)
	if err != nil {
		os.Exit(1)
	}
	err = webhookframe.AddToMgr(mgr, exampleWebhookAdapter)
	if err != nil {
		os.Exit(1)
	}

	if err = mgr.Start(signals.SetupSignalHandler()); err != nil {
		os.Exit(1)
	}
}
