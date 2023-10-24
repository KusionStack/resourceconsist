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

package webhookadpters

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"kusionstack.io/resourceconsist/pkg/adapters/webhookadpters/alibabacloudslb"
	"kusionstack.io/resourceconsist/pkg/webhook_frame"
)

// builtinAdapters init here, builtin adapters' directory name as key, webhookAdapter as value
var builtinAdapters = map[string]webhook_frame.WebhookAdapter{
	"alibabacloudslb": alibabacloudslb.NewWebhookAdapter(),
}

// AddBuiltinAdaptersToMgr adds webhook adapters of given adapterNames to manager
func AddBuiltinAdaptersToMgr(mgr manager.Manager, adapterNames []string) error {
	for _, adapterName := range adapterNames {
		adapter, exist := builtinAdapters[adapterName]
		if !exist {
			return fmt.Errorf("adapterNames contains %s, which is not built-in adapter", adapterName)
		}

		err := webhook_frame.AddToMgr(mgr, adapter)
		if err != nil {
			return fmt.Errorf("add adapter %s to controller failed, err: %s", adapterName, err.Error())
		}
	}
	return nil
}
