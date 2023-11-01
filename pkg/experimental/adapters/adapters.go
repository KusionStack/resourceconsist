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

package adapters

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	controllerframe "kusionstack.io/resourceconsist/pkg/frame/controller"
	webhookframe "kusionstack.io/resourceconsist/pkg/frame/webhook"
)

type AdapterName string

// builtinControllerAdapterNewFuncs init here, builtin adapters' directory name as key, NewReconcileAdapterFunc as value
var builtinControllerAdapterNewFuncs = map[AdapterName]func(c client.Client) (controllerframe.ReconcileAdapter, error){}

// builtinWebhookAdapters init here, builtin adapters' directory name as key, webhookAdapter as value
var builtinWebhookAdapters = map[AdapterName]webhookframe.WebhookAdapter{}

// AddBuiltinControllerAdaptersToMgr adds controller adapters of given adapterNames to manager
func AddBuiltinControllerAdaptersToMgr(mgr manager.Manager, adapterNames []AdapterName) error {
	for _, adapterName := range adapterNames {
		newAdapterFunc, exist := builtinControllerAdapterNewFuncs[adapterName]
		if !exist {
			return fmt.Errorf("adapterNames contains %s, which is not built-in adapter", adapterName)
		}
		adapter, err := newAdapterFunc(mgr.GetClient())
		if err != nil {
			return fmt.Errorf("get adapter %s failed, err: %s", adapterName, err.Error())
		}
		err = controllerframe.AddToMgr(mgr, adapter)
		if err != nil {
			return fmt.Errorf("add adapter %s to controller failed, err: %s", adapterName, err.Error())
		}
	}
	return nil
}

// AddBuiltinWebhookAdaptersToMgr adds webhook adapters of given adapterNames to manager
func AddBuiltinWebhookAdaptersToMgr(mgr manager.Manager, adapterNames []AdapterName) error {
	for _, adapterName := range adapterNames {
		adapter, exist := builtinWebhookAdapters[adapterName]
		if !exist {
			return fmt.Errorf("adapterNames contains %s, which is not built-in adapter", adapterName)
		}

		err := webhookframe.AddToMgr(mgr, adapter)
		if err != nil {
			return fmt.Errorf("add adapter %s to controller failed, err: %s", adapterName, err.Error())
		}
	}
	return nil
}
