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

package controlleradapters

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"kusionstack.io/resourceconsist/pkg/adapters/controlleradapters/alibabacloudslb"
	"kusionstack.io/resourceconsist/pkg/controller_frame"
)

// builtinAdapterNewFuncs init here, builtin adapters' directory name as key, NewReconcileAdapterFunc as value
var builtinAdapterNewFuncs = map[string]func(c client.Client) (controller_frame.ReconcileAdapter, error){
	"alibabacloudslb": alibabacloudslb.NewReconcileAdapter,
}

// AddBuiltinAdaptersToMgr adds controller adapters of given adapterNames to manager
func AddBuiltinAdaptersToMgr(mgr manager.Manager, adapterNames []string) error {
	for _, adapterName := range adapterNames {
		newAdapterFunc, exist := builtinAdapterNewFuncs[adapterName]
		if !exist {
			return fmt.Errorf("adapterNames contains %s, which is not built-in adapter", adapterName)
		}
		adapter, err := newAdapterFunc(mgr.GetClient())
		if err != nil {
			return fmt.Errorf("get adapter %s failed, err: %s", adapterName, err.Error())
		}
		err = controller_frame.AddToMgr(mgr, adapter)
		if err != nil {
			return fmt.Errorf("add adapter %s to controller failed, err: %s", adapterName, err.Error())
		}
	}
	return nil
}
